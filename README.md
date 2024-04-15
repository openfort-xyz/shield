# Shield

![Openfort Protocol][banner-image]

<div align="center">
  <h4>
    <a href="https://www.openfort.xyz/">
      Website
    </a>
    <span> | </span>
    <a href="https://www.openfort.xyz/docs">
      Documentation
    </a>
    <span> | </span>
    <a href="https://www.openfort.xyz/docs/reference/api/authentication">
      API Docs
    </a>
    <span> | </span>
    <a href="https://twitter.com/openfortxyz">
      Twitter
    </a>
  </h4>
</div>

[banner-image]: https://blog-cms.openfort.xyz/uploads/Shield_85938c811e.png



[![Go Report Card](https://goreportcard.com/badge/go.openfort.xyz/shield)](https://goreportcard.com/report/go.openfort.xyz/shield)
## Overview
Shield by Openfort is a secure service dedicated to the protection of sensitive data. It ensures the confidentiality and security of data management by offering a robust framework for storing secrets and encryption parameters.

---
## Project API

The Project API is a secure interface for managing projects in Shield. Each project acts as a container for users and their secrets. The API supports Create, Retrieve, Update, and Delete (CRUD) operations and uses API key and API secret for authentication.

### Endpoints

#### 1. Register a Project
In case you want to encrypt all the shares you can set `generate_encryption_key` to `true` in the request body. This will generate a new encryption key for the project. This key is split into two parts, one part is stored in the database and the other part is returned in the response. With this configuration the entity that hosts shield will not be able to decrypt the shares. If you want to decrypt the shares you will need to provide the encryption key part.
> The encryption key can't be recovered if lost. It is recommended to store the encryption key securely.

- **POST**: `https://shield.openfort.xyz/register`
- **Body**:
  ```json
  { "name": "Test Project", "generate_encryption_key": true }
  ```
- **Example Request**:
  ```shell
  curl --location 'https://shield.openfort.xyz/register' \
  --data '{ "name": "Test Project" }'
  ```
- **Response**:
  ```json
  {
    "id": "ca8dec8b-0794-4f2c-adfa-fb7961f2185a",
    "name": "Test Project",
    "api_key": "1cdfc2a3-767c-49dc-be42-f78e3746ec82",
    "api_secret": "•••••••",
    "encryption_part": "myRhu0uoymTgFE567285c6gunZa8bRtgUBdOWxp96kg="
  }
  ```

#### 2. Get Project Information
- **GET**: `https://shield.openfort.xyz/project`
- **Request Headers**:
    - `x-api-secret`: •••••••
    - `x-api-key`: d2d617ff-dbb6-480d-993f-dc8ac8307617

#### 3. Get Providers
- **GET**: `https://shield.openfort.xyz/project/providers`
- **Request Headers**:
    - `x-api-secret`: •••••••
    - `x-api-key`: d2d617ff-dbb6-480d-993f-dc8ac8307617
- **Example Request**:
  ```shell
  curl --location 'https://shield.openfort.xyz/project/providers' \
  --header 'x-api-secret: •••••••' \
  --header 'x-api-key: d2d617ff-dbb6-480d-993f-dc8ac8307617'
  ```
- **Response**:
  ```json
  {
    "providers": [
      {"provider_id": "bbbdc787-01b6-4a2c-a79f-3968069a5db1", "type": "CUSTOM"},
      {"provider_id": "cd63e8e1-0f55-4540-a44e-b90994553c89", "type": "OPENFORT"}
    ]
  }
  ```

### 4. Delete Provider
- **Endpoint**: Deletes a provider associated with a project.
- **Method**: DELETE
- **URL**: `https://shield.openfort.xyz/project/providers/:provider_id`

#### Request Details

- **Request Headers**:
    - `x-api-secret`: •••••••
    - `x-api-key`: d2d617ff-dbb6-480d-993f-dc8ac8307617
- **Path Variables**:
    - `provider_id`: 74b16efa-a187-491d-906e-55a15f38e28a

- **Example Request**:
```shell
curl --location --request DELETE 'https://shield.openfort.xyz/project/providers/74b16efa-a187-491d-906e-55a15f38e28a' \
--header 'x-api-secret: •••••••' \
--header 'x-api-key: d2d617ff-dbb6-480d-993f-dc8ac8307617'
```
- **Response**: This request does not return a response body.
- **Status Code**:
    - `200 OK`: Successfully deleted the provider.
    - Other status codes indicating error or failure to delete.

#### 5. Get Provider
- **GET**: `https://shield.openfort.xyz/project/providers/:provider_id`
- **Request Headers**:
    - `x-api-secret`: •••••••
    - `x-api-key`: d2d617ff-dbb6-480d-993f-dc8ac8307617
- **Path Variables**:
    - `provider_id`: bbbdc787-01b6-4a2c-a79f-3968069a5db1
- **Example Request**:
  ```shell
  curl --location 'https://shield.openfort.xyz/project/providers/bbbdc787-01b6-4a2c-a79f-3968069a5db1' \
  --header 'x-api-secret: •••••••' \
  --header 'x-api-key: d2d617ff-dbb6-480d-993f-dc8ac8307617'
  ```
- **Response**:
  ```json
  {
    "provider_id": "bbbdc787-01b6-4a2c-a79f-3968069a5db1",
    "type": "CUSTOM",
    "jwk": "https://mydomain/.well-known/jwks.json"
  }
  ```

#### 6. Update Provider
- **PUT**: `https://shield.openfort.xyz/project/providers/:provider_id`
- **Request Headers**:
    - `x-api-secret`: •••••••
    - `x-api-key`: d2d617ff-dbb6-480d-993f-dc8ac8307617
- **Path Variables**:
    - `provider_id`: bbbdc787-01b6-4a2c-a79f-3968069a5db1
- **Body**:
  ```json
  { "jwk": "https://otherdomain/.well-known/jwks.json" }
  ```

#### 7. Create Providers
- **POST**: `https://shield.openfort.xyz/project/providers`
- **Request Headers**:
    - `x-api-secret`: •••••••
    - `x-api-key`: d2d617ff-dbb6-480d-993f-dc8ac8307617
- **Body**:
  ```json
  {
    "providers": {
      "openfort": {
        "publishable_key": "pk_test_505bc088-905e-5a43-b60b-4c37ed1f887a"
      },
      "custom": {
        "jwk": "https://mydomain/.well-known/jwks.json"
      }
    }
  }
  ```
- Custom provider also can work with a PEM and Key type ("rsa" or "ecdsa" or "ed25519")
  ```json
  {
    "providers": {
      "custom": {
        "pem": "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEEVs/o5+uQbTjL3chynL4wXgUg2R9\nq9UU8I5mEovUf86QZ7kOBIjJwqnzD1omageEHWwHdBO6B+dFabmdT9POxg==\n-----END PUBLIC KEY-----",
        "key_type": "ecdsa"
      }
    }
  }
  ```

#### 8. Get Allowed Origins
- **GET**: `https://shield.openfort.xyz/project/allowed-origins`
- **Request Headers**:
    - `x-api-secret`: •••••••
    - `x-api-key`: d2d617ff-dbb6-480d-993f-dc8ac8307617
- **Example Request**:
  ```shell
  curl --location 'https://shield.openfort.xyz/project/allowed-origins' \
  --header 'x-api-secret: •••••••' \
  --header 'x-api-key: d2d617ff-dbb6-480d-993f-dc8ac8307617'
  ```
- **Response**:
  ```json
  { "origins": ["someorigin"] }
  ```

#### 9. Delete Allowed Origin
- **DELETE**: `https://shield.openfort.xyz/project/allowed-origins/:origin`
- **Request Headers**:
    - `x-api-secret`: •••••••
    - `x-api-key`: d2d617ff-dbb6-480d-993f-dc8ac8307617
- **Path Variables**:
    - `origin`: someorigin

#### 10. Add Allowed Origin
- **POST**: `https://shield.openfort.xyz/project/allowed-origins`
- **Request Headers**:
    - `x-api-secret`: •••••••
    - `x-api-key`: d2d617ff-dbb6-480d-993f-dc8ac8307617
- **Body**:
  ```json
  { "origin": "someorigin" }
  ```

#### 11. Encrypt Project Shares
In case you want to encrypt all the shares you can call this endpoint. This will encrypt all the shares that are not encrypted yet.
> This operation is irreversible. Once the shares are encrypted they can't be decrypted.
> The encryption key can't be recovered if lost. It is recommended to store the encryption key securely.
> This operation is only available for projects that have the encryption key generated. You can do it on Project Registration or using the endpoint to generate a new encryption key.
- **POST**: `https://shield.openfort.xyz/project/encrypt`
- **Request Headers**:
    - `x-api-secret`: •••••••
    - `x-api-key`: d2d617ff-dbb6-480d-993f-dc8ac8307617
- **Body**:
    ```json
    { "encryption_part": "myRhu0uoymTgFE567285c6gunZa8bRtgUBdOWxp96kg=" }
    ```

#### 12. Generate Encryption Key
In case you don't set `generate_encryption_key` to `true` in the project registration you can generate a new encryption key using this endpoint. This will generate a new encryption key for the project. This key is split into two parts, one part is stored in the database and the other part is returned in the response. With this configuration the entity that hosts shield will not be able to decrypt the shares. If you want to decrypt the shares you will need to provide the encryption key part.
> The encryption key can't be recovered if lost. It is recommended to store the encryption key securely.
- **POST**: `https://shield.openfort.xyz/project/encryption-key`
- **Request Headers**:
    - `x-api-secret`: •••••••
    - `x-api-key`: d2d617ff-dbb6-480d-993f-dc8ac8307617
- **Example Request**:
  ```shell
  curl --location 'https://shield.openfort.xyz/project/encryption-key' \
    --header 'x-api-secret: •••••••' \
    --header 'x-api-key: d2d617ff-dbb6-480d-993f-dc8ac8307617'
    ```
- **Response**:
  ```json
  {
    "encryption_part": "myRhu0uoymTgFE567285c6gunZa8bRtgUBdOWxp96kg="
  }
  ```
---
## Shares API

The Shares API is part of Shield, dedicated to securely storing and retrieving user-specific secrets.

### Endpoints

#### 1. Create a Share
- **POST**: `https://shield.openfort.xyz/shares`
- **Request Headers**:
    - `Authorization`: Bearer Token
    - `x-auth-provider`: 'openfort' or 'custom'
    - `x-openfort-provider`: 'firebase' // Optional: Only required if x-auth-provider is 'openfort' and using third-party authentication
    - `x-openfort-token-type`: 'idToken' // Optional: Only required if x-auth-provider is 'openfort' and using third-party authentication
- **Body**:
```json
  {
    "secret": "hjkasdhjkladshjkladhjskladhjskl",
    "entropy": "none",
  }
``` 

OR

```json  
  {
    "secret": "hjkasdhjkladshjkladhjskladhjskl",
    "entropy": "user",
    "salt": "somesalt",
    "iterations": 1000,
    "length": 8,
    "digest": "SHA-256"
  }
```  

OR 

```json  
  {
    "secret: "hjkasdhjkladshjkladhjskladhjskl",
    "entropy": "project",
    "encryption_part": "myRhu0uoymTgFE567285c6gunZa8bRtgUBdOWxp96kg="
  }
  ```
#### 2. Get Share
- **GET**: `https://shield.openfort.xyz/shares`
- **Request Headers**:
    - `Authorization`: Bearer Token
    - `x-auth-provider`: 'openfort' or 'custom'
    - `x-openfort-provider`: 'firebase' // Optional: Only required if x-auth-provider is 'openfort' and using third-party authentication
    - `x-openfort-token-type`: 'idToken' // Optional: Only required if x-auth-provider is 'openfort' and using third-party authentication
    - `x-encryption-part`: 'myRhu0uoymTgFE567285c6gunZa8bRtgUBdOWxp96kg=' // Optional: Only required if the share have project entropy
- **Example Request**:
  ```shell
  curl --location 'https://shield.openfort.xyz/shares' \
  --header 'Authorization: Bearer •••••••' \
  --header 'x-auth-provider: openfort' \
  --header 'x-openfort-provider: firebase' \
  --header 'x-openfort-token-type: idToken' \
  --header 'x-api-key: d2d617ff-dbb6-480d-993f-dc8ac8307617'
  ```
- **Response**:
  ```json
  {
    "secret": "hjkasdhjkladshjkladhjskladhjskl",
    "entropy": "none"
  }
  ```

#### 3. Delete Share
- **DELETE**: `https://shield.openfort.xyz/shares`
- **Request Headers**:
  - `Authorization`: Bearer Token
  - `x-auth-provider`: 'openfort' or 'custom'
  - `x-openfort-provider`: 'firebase' // Optional: Only required if x-auth-provider is 'openfort' and using third-party authentication
  - `x-openfort-token-type`: 'idToken' // Optional: Only required if x-auth-provider is 'openfort' and using third-party authentication
  - `x-encryption-part`: 'myRhu0uoymTgFE567285c6gunZa8bRtgUBdOWxp96kg=' // Optional: Only required if the share have project entropy
- **Example Request**:
  ```shell
  curl --location --request DELETE 'https://shield.openfort.xyz/shares' \
  --header 'Authorization: Bearer •••••••' \
  --header 'x-auth-provider: openfort' \
  --header 'x-openfort-provider: firebase' \
  --header 'x-openfort-token-type: idToken' \
  --header 'x-api-key: d2d617ff-dbb6-480d-993f-dc8ac8307617'
  ```
