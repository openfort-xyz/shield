package sharehdl

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	ua "github.com/mileusna/useragent"
	"go.openfort.xyz/shield/internal/adapters/handlers/rest/api"
	"go.openfort.xyz/shield/internal/applications/shareapp"
	"go.openfort.xyz/shield/pkg/logger"
)

type Handler struct {
	app       *shareapp.ShareApplication
	logger    *slog.Logger
	parser    *parser
	validator *validator
}

func New(app *shareapp.ShareApplication) *Handler {
	return &Handler{
		app:       app,
		logger:    logger.New("share_handler"),
		parser:    newParser(),
		validator: newValidator(),
	}
}

func getDetailedDeviceName(userAgent ua.UserAgent) string {
	if len(userAgent.Device) == 0 {
		if userAgent.Mobile {
			return "Mobile"
		}
		if userAgent.Desktop {
			return "Desktop"
		}
		if userAgent.Tablet {
			return "Tablet"
		}
		return "unknown"
	}
	return userAgent.Device
}

func parseUserAgent(r *http.Request) (*PasskeyEnv, *api.Error) {
	uaHeaders := r.Header["User-Agent"]
	if len(uaHeaders) > 1 {
		return nil, api.ErrBadRequestWithMessage("header User-Agent needs to be defined exactly once")
	}

	if len(uaHeaders) == 1 {
		parsedUaHeader := ua.Parse(uaHeaders[0])
		deviceName := getDetailedDeviceName(parsedUaHeader)
		ret := PasskeyEnv{
			Name:      &parsedUaHeader.Name,
			OS:        &parsedUaHeader.OS,
			OSVersion: &parsedUaHeader.OSVersion,
			Device:    &deviceName,
		}
		return &ret, nil
	}

	return nil, nil
}

// Keychain gets the keychain
// @Summary Get keychain
// @Description Get the keychain for the user
// @Tags Share
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param Authorization header string true "
// @Param X-Auth-Provider header string true "Auth Provider"
// @Param reference query string false "Reference"
// @Success 200 {object} KeychainResponse "Successful response"
// @Failure 404 "Description: Not Found"
// @Failure 500 "Description: Internal Server Error"
// @Router /shares/keychain [get]
func (h *Handler) Keychain(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "getting keychain")
	referenceQuery := r.URL.Query().Get("reference")
	var reference *string
	if referenceQuery != "" {
		reference = &referenceQuery
	}

	var opts []shareapp.Option
	encryptionPart := r.Header.Get(EncryptionPartHeader)
	if encryptionPart != "" {
		opts = append(opts, shareapp.WithEncryptionPart(encryptionPart))
	}

	encryptionSession := r.Header.Get(EncryptionSessionHeader)
	if encryptionSession != "" {
		opts = append(opts, shareapp.WithEncryptionSession(encryptionSession))
	}

	keychain, err := h.app.GetKeychainShares(ctx, reference, opts...)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	var response KeychainResponse
	for _, share := range keychain {
		response.Shares = append(response.Shares, h.parser.fromDomain(share))
	}

	resp, err := json.Marshal(response)
	if err != nil {
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

// Get share by its reference
// @Summary Get share
// @Description Get the share for the user
// @Tags Share
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param Authorization header string true "
// @Param X-Auth-Provider header string true "Auth Provider"
// @Param reference query string false "Reference"
// @Success 200 {object} GetShareResponse "Successful response"
// @Failure 404 "Description: Not Found"
// @Failure 500 "Description: Internal Server Error"
// @Router /shares [get]
func (h *Handler) GetShareByReference(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "getting share by reference")
	var reference string

	arg := mux.Vars(r)["reference"]
	if arg != "" {
		reference = arg
	}

	var opts []shareapp.Option
	encryptionPart := r.Header.Get(EncryptionPartHeader)
	if encryptionPart != "" {
		opts = append(opts, shareapp.WithEncryptionPart(encryptionPart))
	}

	encryptionSession := r.Header.Get(EncryptionSessionHeader)
	if encryptionSession != "" {
		opts = append(opts, shareapp.WithEncryptionSession(encryptionSession))
	}

	share, err := h.app.GetShareByReference(ctx, reference, opts...)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	resp, err := json.Marshal(GetShareResponse(*h.parser.fromDomain(share)))
	if err != nil {
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

// RegisterShare registers a new share
// @Summary Register new share
// @Description Register a new share for the user
// @Tags Share
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param Authorization header string true "Bearer token"
// @Param X-Auth-Provider header string true "Auth Provider"
// @Param X-Openfort-Provider header string false "Openfort Provider"
// @Param X-Openfort-Token-Type header string false "Openfort Token Type"
// @Param registerShareRequest body RegisterShareRequest true "Register Share Request"
// @Success 201 "Description: Share registered successfully"
// @Failure 400 {object} api.Error "Bad Request"
// @Failure 404 {object} api.Error "Not Found"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /shares/register [post]
func (h *Handler) RegisterShare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "registering share")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to read request body"))
		return
	}

	var req RegisterShareRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to parse request body"))
		return
	}

	if errV := h.validator.validateShare((*Share)(&req)); errV != nil {
		api.RespondWithError(w, errV)
		return
	}

	if req.PasskeyReference != nil && req.PasskeyReference.PasskeyEnv == nil {
		sourceEnv, apiErr := parseUserAgent(r)
		if apiErr != nil {
			api.RespondWithError(w, apiErr)
			return
		}
		req.PasskeyReference.PasskeyEnv = sourceEnv
	}

	share := h.parser.toDomain((*Share)(&req))
	var opts []shareapp.Option
	if req.EncryptionPart != "" {
		opts = append(opts, shareapp.WithEncryptionPart(req.EncryptionPart))
	}
	if req.EncryptionSession != "" {
		opts = append(opts, shareapp.WithEncryptionSession(req.EncryptionSession))
	}
	err = h.app.RegisterShare(ctx, share, opts...)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// UpdateShare updates a share
// @Summary Update share
// @Description Update a share for the user
// @Tags Share
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param Authorization header string true "Bearer token"
// @Param X-Auth-Provider header string true "Auth Provider"
// @Param X-Openfort-Provider header string false "Openfort Provider"
// @Param X-Openfort-Token-Type header string false "Openfort Token Type"
// @Param updateShareRequest body UpdateShareRequest true "Update Share Request"
// @Success 200 {object} UpdateShareResponse "Successful response"
// @Failure 400 {object} api.Error "Bad Request"
// @Failure 404 {object} api.Error "Not Found"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /shares [put]
func (h *Handler) UpdateShare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "updating share")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to read request body"))
		return
	}

	var req UpdateShareRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to parse request body"))
		return
	}

	if errV := h.validator.validateShare((*Share)(&req)); errV != nil {
		api.RespondWithError(w, errV)
		return
	}

	if req.PasskeyReference != nil && req.PasskeyReference.PasskeyEnv == nil {
		sourceEnv, apiErr := parseUserAgent(r)
		if apiErr != nil {
			api.RespondWithError(w, apiErr)
			return
		}
		req.PasskeyReference.PasskeyEnv = sourceEnv
	}

	share := h.parser.toDomain((*Share)(&req))
	var opts []shareapp.Option
	if req.EncryptionPart != "" {
		opts = append(opts, shareapp.WithEncryptionPart(req.EncryptionPart))
	}
	if req.EncryptionSession != "" {
		opts = append(opts, shareapp.WithEncryptionSession(req.EncryptionSession))
	}
	shr, err := h.app.UpdateShare(ctx, share, req.Reference, opts...)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	resp, err := json.Marshal(UpdateShareResponse(*h.parser.fromDomain(shr)))
	if err != nil {
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

// DeleteShare deletes a share
// @Summary Delete share
// @Description Delete a share for the user
// @Tags Share
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param Authorization header string true "Bearer token"
// @Param X-Auth-Provider header string true "Auth Provider"
// @Param X-Openfort-Provider header string false "Openfort Provider"
// @Param X-Openfort-Token-Type header string false "Openfort Token Type"
// @Success 204 "Description: Share deleted successfully"
// @Failure 404 "Description: Not Found"
// @Failure 500 "Description: Internal Server Error"
// @Router /shares [delete]
func (h *Handler) DeleteShare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "deleting share")

	var reference *string

	arg := mux.Vars(r)["reference"]
	if arg != "" {
		reference = &arg
	}

	err := h.app.DeleteShare(ctx, reference)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetShare gets a share
// @Summary Get share
// @Description Get a share for the user
// @Tags Share
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param Authorization header string true "Bearer token"
// @Param X-Auth-Provider header string true "Auth Provider"
// @Param X-Openfort-Provider header string false "Openfort Provider"
// @Param X-Openfort-Token-Type header string false "Openfort Token Type"
// @Param X-Encryption-Part header string false "Encryption Part"
// @Success 200 {object} GetShareResponse "Successful response"
// @Failure 404 "Description: Not Found"
// @Failure 500 "Description: Internal Server Error"
// @Router /shares [get]
func (h *Handler) GetShare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "getting share")

	var opts []shareapp.Option
	encryptionPart := r.Header.Get(EncryptionPartHeader)
	if encryptionPart != "" {
		opts = append(opts, shareapp.WithEncryptionPart(encryptionPart))
	}

	encryptionSession := r.Header.Get(EncryptionSessionHeader)
	if encryptionSession != "" {
		opts = append(opts, shareapp.WithEncryptionSession(encryptionSession))
	}

	shr, err := h.app.GetShare(ctx, opts...)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	resp, err := json.Marshal(GetShareResponse(*h.parser.fromDomain(shr)))
	if err != nil {
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

// GetShareEncryption gets the encryption of a share
// @Summary Get share encryption
// @Description Get the encryption of a share for the user
// @Tags Share
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param Authorization header string true "Bearer token"
// @Param X-Auth-Provider header string true "Auth Provider"
// @Param X-Openfort-Provider header string false "Openfort Provider"
// @Param X-Openfort-Token-Type header string false "Openfort Token Type"
// @Success 200 {object} GetShareEncryptionResponse "Successful response"
// @Failure 404 "Description: Not Found"
// @Failure 500 "Description: Internal Server Error"
// @Router /shares/encryption [get]
func (h *Handler) GetShareEncryption(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "getting share")

	shareEntropy, encryptionParameters, err := h.app.GetShareEncryption(ctx)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	encryptionResponse := GetShareEncryptionResponse{Entropy: h.parser.mapDomainEntropy[shareEntropy]}

	// EntropyUser -> include crypto info (digest, iterations, salt, length)
	if encryptionResponse.Entropy == EntropyUser {
		encryptionResponse.Digest = &encryptionParameters.Digest
		encryptionResponse.Iterations = &encryptionParameters.Iterations
		encryptionResponse.Length = &encryptionParameters.Length
		encryptionResponse.Salt = &encryptionParameters.Salt
	}
	// Implicit "else-do-nothing", project entropy is self explanatory and NoneEntropy has no config fields
	// to take into account

	resp, err := json.Marshal(encryptionResponse)
	if err != nil {
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

// GetSharesEncryptionForReferences pairs all the given references to their corresponding
// source of entropy (i.e. none|user|project|passkey)
// @Summary Get shares entropy sources
// @Description Get shares entropy sources for given references
// @Tags Share Entropy
// @Success 200 {map} Reference -> ShareEncryptionDetails
// @Failure 400 Bad Request
// @Failure 500 Internal Server Error
// @Router /shares/encryption/reference/bulk
func (h *Handler) GetSharesEncryptionForReferences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "getting share storage methods")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to read request body"))
		return
	}

	var requestedReferences GetSharesEncryptionForReferencesRequest
	err = json.Unmarshal(body, &requestedReferences)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to parse request body"))
		return
	}

	if len(requestedReferences.References) > MaxBulkSize {
		api.RespondWithError(w, api.ErrBadRequestWithMessage(fmt.Sprintf("Requests with more than %d elements are not allowed", MaxBulkSize)))
		return
	}

	// We're not failing if some of the references are invalid but just opting to return a "not-found" status if a reference
	// a) doesn't exist
	// b) it exists but it's tied to a share that doesn't belong to the same project
	// Also notice that this endpoint DOES return an entry for every requested asset, not only the existing ones
	// This way, the response will still be exhaustive whilst making sure we're not giving too much extra information
	// away for free
	domainEncryptionTypes, err := h.app.GetSharesEncryptionForReferences(ctx, requestedReferences.References)

	if err != nil {
		// Any error here must be the server's fault (the request is well-formed)
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	var responseBody GetSharesEncryptionForReferencesResponse
	responseBody.EncryptionTypes = map[string]EncryptionTypeResponse{}

	// We're iterating based on the REQUESTED references
	// And checking against the obtained encryption types in the backend
	// As we mentioned before, the domain doesn't know about response/size limit requirements
	// Those belong to the transport/presentation layer
	for _, requestedReference := range requestedReferences.References {
		val, exists := domainEncryptionTypes[requestedReference]
		if exists {
			encryptionType := h.parser.mapDomainEntropy[val.Entropy]
			responseBody.EncryptionTypes[requestedReference] = EncryptionTypeResponse{
				Status:         EncryptionTypeStatusFound,
				EncryptionType: &encryptionType,
				PasskeyID:      val.PasskeyID,
				PasskeyEnv:     h.parser.toPasskeyEnv(val.PasskeyEnv),
			}
		} else {
			responseBody.EncryptionTypes[requestedReference] = EncryptionTypeResponse{
				Status: EncryptionTypeStatusNotFound,
			}
		}
	}

	// responseBody is univocally correct once we reached this point
	w.WriteHeader(http.StatusOK)
	resp, _ := json.Marshal(responseBody)
	_, _ = w.Write(resp)
}

// GetSharesEncryptionForUsers pairs all the given users to their corresponding
// source of entropy (i.e. none|user|project|passkey)
// ⚠️ This endpoint only makes sense if the requested accounts are LEGACY so their share reference is default
// ⚠️ Otherwise the requester will get an arbitrary share since newly created accounts can hold multiple shares
// and it's guaranteed that the share-userID mapping will be 1on1
// @Summary Get shares entropy sources
// @Description Get shares entropy sources for given (external) user IDs
// @Tags Share Entropy
// @Success 200 {map} UserID -> ShareEncryptionDetails
// @Failure 400 Bad Request
// @Failure 500 Internal Server Error
// @Router /shares/encryption/user/bulk
func (h *Handler) GetSharesEncryptionForUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "getting share storage methods")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to read request body"))
		return
	}

	var requestedUsers GetSharesEncryptionForUsersRequest
	err = json.Unmarshal(body, &requestedUsers)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to parse request body"))
		return
	}

	if len(requestedUsers.UserIDs) > MaxBulkSize {
		api.RespondWithError(w, api.ErrBadRequestWithMessage(fmt.Sprintf("Requests with more than %d elements are not allowed", MaxBulkSize)))
		return
	}

	// We're not failing if some of the users are invalid but just opting to return a "not-found" status if a user
	// a) doesn't exist
	// b) it exists but it's tied to a share that doesn't belong to the same project
	// Also notice that this endpoint DOES return an entry for every requested asset, not only the existing ones
	// This way, the response will still be exhaustive whilst making sure we're not giving too much extra information
	// away for free
	domainEncryptionTypes, err := h.app.GetSharesEncryptionForUsers(ctx, requestedUsers.UserIDs, requestedUsers.Reference)

	if err != nil {
		// Any error here must be the server's fault (the request is well-formed)
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	var responseBody GetSharesEncryptionForUsersResponse
	responseBody.EncryptionTypes = map[string]EncryptionTypeResponse{}

	// We're iterating based on the REQUESTED users
	// And checking against the obtained encryption types in the backend
	// As we mentioned before, the domain doesn't know about response/size limit requirements
	// Those belong to the transport/presentation layer
	for _, requestedUser := range requestedUsers.UserIDs {
		val, exists := domainEncryptionTypes[requestedUser]
		if exists {
			encryptionType := h.parser.mapDomainEntropy[val.Entropy]
			responseBody.EncryptionTypes[requestedUser] = EncryptionTypeResponse{
				Status:         EncryptionTypeStatusFound,
				EncryptionType: &encryptionType,
				PasskeyID:      val.PasskeyID,
				PasskeyEnv:     h.parser.toPasskeyEnv(val.PasskeyEnv),
			}
		} else {
			responseBody.EncryptionTypes[requestedUser] = EncryptionTypeResponse{
				Status: EncryptionTypeStatusNotFound,
			}
		}
	}

	// responseBody is univocally correct once we reached this point
	w.WriteHeader(http.StatusOK)
	resp, _ := json.Marshal(responseBody)
	_, _ = w.Write(resp)
}

// GetShareStorageMethods list the available share storage methods
// @Summary Get share storage methods
// @Description Get the available share storage methods
// @Tags Share
// @Produce json
// @Success 200 {array} ShareStorageMethod "Successful response"
// @Failure 500 "Description: Internal Server Error"
// @Router /shares/storage-methods [get]
func (h *Handler) GetShareStorageMethods(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "getting share storage methods")

	storageMethods, err := h.app.GetShareStorageMethods(ctx)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	var shareStorageMethodJsons []*ShareStorageMethod
	for _, method := range storageMethods {
		shareStorageMethodJsons = append(shareStorageMethodJsons, h.parser.fromDomainShareStorageMethod(method))
	}

	response := GetShareStorageMethodsResponse{
		Methods: shareStorageMethodJsons,
	}

	resp, err := json.Marshal(response)
	if err != nil {
		api.RespondWithError(w, api.ErrInternal)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}
