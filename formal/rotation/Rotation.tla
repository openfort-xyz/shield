---------------------------- MODULE Rotation ----------------------------
(***************************************************************************)
(* Two-phase rotation of a Shield project's encryption key, API key and  *)
(* API secret,                                                             *)
(* racing share writers, the shamir migration job, stale project parts     *)
(* (configs and encryption sessions) and lost HTTP responses.              *)
(*                                                                         *)
(* Abstraction. A key is an id. Splitting key k yields stored part k and   *)
(* project part k. Shamir combine accepts ANY pair of well-formed parts,   *)
(* so combining parts of different splits yields Garbage: a key nobody     *)
(* can reproduce. A share encrypted under Garbage is lost.                 *)
(*                                                                         *)
(* Atomicity. Each action is one database transaction. The rotation       *)
(* commit holds FOR UPDATE on the part row; a share write re-reads the     *)
(* part under FOR SHARE in the same transaction as its INSERT/UPDATE.      *)
(* Postgres READ COMMITTED makes the blocked FOR SHARE return the          *)
(* committed row (sharerepo TestPartRowLockSemantics), so the pair         *)
(* serializes and each is modeled as one atomic step. The race is the      *)
(* gap between a writer's first (unlocked) read and its persist.           *)
(***************************************************************************)
EXTENDS Naturals, FiniteSets

CONSTANTS
    Writers,          \* share writers; writer w owns share w
    MaxKey,           \* bound on generated encryption keys and API keys
    MaxSec,           \* bound on generated API secrets
    MutNoRecheck,     \* persist without re-reading the part under FOR SHARE
    MutDeleteInsert,  \* commit replaces the part row; guard treats "no row" as no conflict
    MutNoKeyCheck,    \* builders skip the stored key-check verification
    MutMultiPending,  \* prepare adds a pending key instead of replacing it
    MutOneShot,       \* single-call rotation: new part only in the commit response
    MutCommitNoAuth,  \* API-key commit accepted with the current key
    MutNoBackfill,    \* prepare leaves a legacy project without a key check
    MutSecCommitNoAuth \* API-secret commit accepted with the current secret

None == 0
Garbage == MaxKey + 1
Legacy == MaxKey + 2          \* no key check stored (pre-check projects)

Recon(stored, projectPart) == IF stored = projectPart THEN stored ELSE Garbage

VARIABLES
    cur,        \* id of the current stored part (= current key)
    curCheck,   \* key check of cur, or Legacy
    gen,        \* part-row identity; changes only under MutDeleteInsert
    pend,       \* pending encryption keys (correct spec: at most one)
    nextKey,
    shares,     \* Writers -> key the share's ciphertext is under, None = absent
    everHeld,   \* every project part the client ever held: stale configs and sessions
    cpart,      \* client's working project part
    cnew,       \* part received from prepare, not yet confirmed committed
    wpc, wkey, wread, wgen,
    spc, skey, sread, sgen, shamirDone,
    apiKey, pendKey, ck, cpend, nextApi,
    apiSec, pendSec, cs, cpendSec, nextSec

encVars == <<cur, curCheck, gen, pend, nextKey>>
clientVars == <<everHeld, cpart, cnew>>
writerVars == <<wpc, wkey, wread, wgen>>
shamirVars == <<spc, skey, sread, sgen, shamirDone>>
apiVars == <<apiKey, pendKey, ck, cpend, nextApi>>
secVars == <<apiSec, pendSec, cs, cpendSec, nextSec>>
vars == <<encVars, shares, clientVars, writerVars, shamirVars, apiVars, secVars>>

Init ==
    /\ cur = 1 /\ curCheck = Legacy /\ gen = 0 /\ pend = {} /\ nextKey = 2
    /\ shares = [w \in Writers |-> IF w = 1 THEN 1 ELSE None]
    /\ everHeld = {1} /\ cpart = 1 /\ cnew = None
    /\ wpc = [w \in Writers |-> "idle"] /\ wkey = [w \in Writers |-> None]
    /\ wread = [w \in Writers |-> None] /\ wgen = [w \in Writers |-> 0]
    /\ spc = "idle" /\ skey = None /\ sread = None /\ sgen = 0 /\ shamirDone = FALSE
    /\ apiKey = 1 /\ pendKey = None /\ ck = 1 /\ cpend = None /\ nextApi = 2
    /\ apiSec = 1 /\ pendSec = None /\ cs = 1 /\ cpendSec = None /\ nextSec = 2

\* The builders' Build(): reconstruct, then verify against the key check.
KeyAccepted(k) == curCheck = Legacy \/ MutNoKeyCheck \/ k = curCheck

\* sharerepo.underGuard: SELECT part ... FOR SHARE, compare, else abort.
GuardPasses(read, readGen) ==
    IF MutNoRecheck THEN TRUE
    ELSE IF readGen # gen THEN MutDeleteInsert
    ELSE read = cur

\* Old-part proof: the check value, or for legacy projects every share decrypts.
OldPartValid(p) ==
    LET k == Recon(cur, p) IN
    IF curCheck = Legacy
    THEN /\ \E s \in Writers : shares[s] # None        \* else unverifiable: 409 EC_UNVERIFIABLE
         /\ \A s \in Writers : shares[s] \in {None, k}
    ELSE k = curCheck

-----------------------------------------------------------------------------
(* Share writers: RegisterShare / UpdateShare through reconstructEncryptionKey *)

WriterRead(w) ==
    /\ wpc[w] = "idle"
    /\ \E p \in everHeld :
         /\ KeyAccepted(Recon(cur, p))
         /\ wkey' = [wkey EXCEPT ![w] = Recon(cur, p)]
    /\ wpc' = [wpc EXCEPT ![w] = "ready"]
    /\ wread' = [wread EXCEPT ![w] = cur]
    /\ wgen' = [wgen EXCEPT ![w] = gen]
    /\ UNCHANGED <<encVars, shares, clientVars, shamirVars, apiVars, secVars>>

WriterPersist(w) ==
    /\ wpc[w] = "ready"
    /\ shares' = IF GuardPasses(wread[w], wgen[w])
                 THEN [shares EXCEPT ![w] = wkey[w]]
                 ELSE shares                       \* 409 EC_ROTATED
    /\ wpc' = [wpc EXCEPT ![w] = "idle"]
    /\ UNCHANGED <<encVars, clientVars, wkey, wread, wgen, shamirVars, apiVars, secVars>>

(* shamirjob.Execute: re-encrypts every share of a legacy project, BulkUpdate *)

ShamirRead ==
    /\ ~shamirDone /\ spc = "idle"
    /\ Recon(cur, cpart) = cur
    /\ spc' = "ready" /\ skey' = cur /\ sread' = cur /\ sgen' = gen
    /\ UNCHANGED <<encVars, shares, clientVars, writerVars, shamirDone, apiVars, secVars>>

ShamirPersist ==
    /\ spc = "ready"
    /\ shares' = IF GuardPasses(sread, sgen)
                 THEN [s \in Writers |-> IF shares[s] = None THEN None ELSE skey]
                 ELSE shares
    /\ spc' = "idle" /\ shamirDone' = TRUE
    /\ UNCHANGED <<encVars, clientVars, writerVars, skey, sread, sgen, apiVars, secVars>>

-----------------------------------------------------------------------------
(* Encryption-key rotation: prepare / commit *)

RotPrepare ==
    /\ ~MutOneShot
    /\ nextKey <= MaxKey
    /\ OldPartValid(cpart)
    /\ pend' = IF MutMultiPending THEN pend \cup {nextKey} ELSE {nextKey}
    /\ nextKey' = nextKey + 1
    \* Backfill: once a new part can leave the server, a part that does not
    \* match the stored one must be detectable, so a legacy project gets the
    \* check of the key prepare just verified.
    /\ curCheck' = IF curCheck = Legacy /\ ~MutNoBackfill THEN Recon(cur, cpart) ELSE curCheck
    /\ \E delivered \in BOOLEAN :
         IF delivered
         THEN /\ cnew' = nextKey /\ everHeld' = everHeld \cup {nextKey}
         ELSE UNCHANGED <<cnew, everHeld>>
    /\ UNCHANGED <<cur, gen, shares, cpart, writerVars, shamirVars, apiVars, secVars>>

ConfirmNew(delivered) ==
    IF delivered THEN cpart' = cnew /\ cnew' = None ELSE UNCHANGED <<cpart, cnew>>

DropNew(delivered) ==
    IF delivered THEN cnew' = None /\ UNCHANGED cpart ELSE UNCHANGED <<cpart, cnew>>

RotCommit ==
    /\ ~MutOneShot
    /\ cnew # None
    /\ \E delivered \in BOOLEAN :
         CASE curCheck # Legacy /\ Recon(cur, cnew) = curCheck ->      \* 200 already_committed
                /\ ConfirmNew(delivered)
                /\ UNCHANGED <<encVars, shares, everHeld>>
           [] pend = {} \/ cnew \notin pend ->                          \* 409 no pending / expired, 400 stale pending
                /\ DropNew(delivered)
                /\ UNCHANGED <<encVars, shares, everHeld>>
           [] ~OldPartValid(cpart) ->                                   \* 400 old part wrong
                /\ UNCHANGED <<encVars, shares, clientVars>>
           [] OTHER ->                                                  \* 200 committed
                /\ shares' = [s \in Writers |-> IF shares[s] = None THEN None ELSE cnew]
                /\ cur' = cnew /\ curCheck' = cnew /\ pend' = {}
                /\ gen' = IF MutDeleteInsert THEN gen + 1 ELSE gen
                /\ UNCHANGED nextKey
                /\ ConfirmNew(delivered)
                /\ UNCHANGED everHeld
    /\ UNCHANGED <<writerVars, shamirVars, apiVars, secVars>>

RotOneShot ==
    /\ MutOneShot
    /\ nextKey <= MaxKey
    /\ OldPartValid(cpart)
    /\ shares' = [s \in Writers |-> IF shares[s] = None THEN None ELSE nextKey]
    /\ cur' = nextKey /\ curCheck' = nextKey /\ nextKey' = nextKey + 1
    /\ \E delivered \in BOOLEAN :
         IF delivered
         THEN cpart' = nextKey /\ everHeld' = everHeld \cup {nextKey}
         ELSE UNCHANGED <<cpart, everHeld>>
    /\ UNCHANGED <<gen, pend, cnew, writerVars, shamirVars, apiVars, secVars>>

PendingExpires ==
    /\ pend # {}
    /\ pend' = {}
    /\ UNCHANGED <<cur, curCheck, gen, nextKey, shares, clientVars, writerVars, shamirVars, apiVars, secVars>>

-----------------------------------------------------------------------------
(* API-key rotation: prepare (current key) / commit (pending key proves receipt) *)

ApiPrepare ==
    /\ nextApi <= MaxKey
    /\ ck = apiKey /\ cs = apiSec
    /\ pendKey' = nextApi /\ nextApi' = nextApi + 1
    /\ \E delivered \in BOOLEAN :
         IF delivered THEN cpend' = nextApi ELSE UNCHANGED cpend
    /\ UNCHANGED <<apiKey, ck, encVars, shares, clientVars, writerVars, shamirVars, secVars>>

ApiCommit ==
    LET key == IF cpend # None THEN cpend ELSE ck IN
    /\ \E delivered \in BOOLEAN :
         CASE cs # apiSec ->                                            \* 401: secret rotated meanwhile
                /\ IF delivered THEN cpend' = None ELSE UNCHANGED cpend
                /\ UNCHANGED <<apiKey, pendKey, ck>>
           [] pendKey # None /\ key = pendKey ->                        \* 200 committed
                /\ apiKey' = pendKey /\ pendKey' = None
                /\ IF delivered THEN ck' = key /\ cpend' = None ELSE UNCHANGED <<ck, cpend>>
           [] key = apiKey /\ (pendKey = None) ->                       \* 200 current
                /\ IF delivered THEN ck' = key /\ cpend' = None ELSE UNCHANGED <<ck, cpend>>
                /\ UNCHANGED <<apiKey, pendKey>>
           [] key = apiKey /\ MutCommitNoAuth ->                        \* mutant: swap on current key
                /\ apiKey' = pendKey /\ pendKey' = None
                /\ IF delivered THEN ck' = pendKey /\ cpend' = None ELSE UNCHANGED <<ck, cpend>>
           [] key = apiKey ->                                           \* 409 pending not proven
                UNCHANGED <<apiKey, pendKey, ck, cpend>>
           [] OTHER ->                                                  \* 401: pending key dead
                /\ IF delivered THEN cpend' = None ELSE UNCHANGED cpend
                /\ UNCHANGED <<apiKey, pendKey, ck>>
    /\ UNCHANGED <<nextApi, encVars, shares, clientVars, writerVars, shamirVars, secVars>>

ApiPendingExpires ==
    /\ pendKey # None
    /\ pendKey' = None
    /\ UNCHANGED <<apiKey, ck, cpend, nextApi, encVars, shares, clientVars, writerVars, shamirVars, secVars>>

-----------------------------------------------------------------------------
(* API-secret rotation: prepare (current key and secret) / commit (current *)
(* key with the pending secret proves receipt). The commit decides under   *)
(* the project row lock that API-key commits also take, so each action is  *)
(* atomic against them.                                                    *)

SecPrepare ==
    /\ nextSec <= MaxSec
    /\ ck = apiKey /\ cs = apiSec
    /\ pendSec' = nextSec /\ nextSec' = nextSec + 1
    /\ \E delivered \in BOOLEAN :
         IF delivered THEN cpendSec' = nextSec ELSE UNCHANGED cpendSec
    /\ UNCHANGED <<apiSec, cs, encVars, shares, clientVars, writerVars, shamirVars, apiVars>>

SecCommit ==
    LET sec == IF cpendSec # None THEN cpendSec ELSE cs IN
    /\ \E delivered \in BOOLEAN :
         CASE ck # apiKey ->                                            \* 401: key rotated meanwhile
                /\ IF delivered THEN cpendSec' = None ELSE UNCHANGED cpendSec
                /\ UNCHANGED <<apiSec, pendSec, cs>>
           [] pendSec # None /\ sec = pendSec ->                        \* 200 committed
                /\ apiSec' = pendSec /\ pendSec' = None
                /\ IF delivered THEN cs' = sec /\ cpendSec' = None ELSE UNCHANGED <<cs, cpendSec>>
           [] sec = apiSec /\ pendSec = None ->                         \* 200 current
                /\ IF delivered THEN cs' = sec /\ cpendSec' = None ELSE UNCHANGED <<cs, cpendSec>>
                /\ UNCHANGED <<apiSec, pendSec>>
           [] sec = apiSec /\ MutSecCommitNoAuth ->                     \* mutant: swap on current secret
                /\ apiSec' = pendSec /\ pendSec' = None
                /\ UNCHANGED <<cs, cpendSec>>
           [] sec = apiSec ->                                           \* 409 pending not proven
                UNCHANGED <<apiSec, pendSec, cs, cpendSec>>
           [] OTHER ->                                                  \* 401/409: pending secret dead
                /\ IF delivered THEN cpendSec' = None ELSE UNCHANGED cpendSec
                /\ UNCHANGED <<apiSec, pendSec, cs>>
    /\ UNCHANGED <<nextSec, encVars, shares, clientVars, writerVars, shamirVars, apiVars>>

SecPendingExpires ==
    /\ pendSec # None
    /\ pendSec' = None
    /\ UNCHANGED <<apiSec, cs, cpendSec, nextSec, encVars, shares, clientVars, writerVars, shamirVars, apiVars>>

-----------------------------------------------------------------------------

Next ==
    \/ \E w \in Writers : WriterRead(w) \/ WriterPersist(w)
    \/ ShamirRead \/ ShamirPersist
    \/ RotPrepare \/ RotCommit \/ RotOneShot \/ PendingExpires
    \/ ApiPrepare \/ ApiCommit \/ ApiPendingExpires
    \/ SecPrepare \/ SecCommit \/ SecPendingExpires

Spec == Init /\ [][Next]_vars

-----------------------------------------------------------------------------
(* Properties *)

\* Every stored project share decrypts under the current key.
NoUndecryptableShare == \A s \in Writers : shares[s] \in {None, cur}

AtMostOnePending == Cardinality(pend) <= 1

\* A client that keeps what it was sent can always reach the current key.
ClientCanRecover == cur \in {cpart, cnew}

\* A client that keeps what it was sent can always authenticate: with a key
\* and a secret it holds.
ApiClientCanAuth == apiKey \in {ck, cpend} /\ apiSec \in {cs, cpendSec}

=============================================================================
