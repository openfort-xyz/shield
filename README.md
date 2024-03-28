# Shield
## Overview
Shield by Openfort is a secure service dedicated to the protection of sensitive data. It ensures the confidentiality and security of data management by offering a robust framework for storing secrets and encryption parameters.

---
## Project API

The Project API is a secure interface for managing projects in Shield. Each project acts as a container for users and their secrets. The API supports Create, Retrieve, Update, and Delete (CRUD) operations and uses API key and API secret for authentication.

### Endpoints

#### 1. Register a Project
- **POST**: `https://shield.openfort.xyz/register`
- **Body**:
  ```json
  { "name": "Test Project" }
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
    "api_secret": "•••••••"
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

---
## Shares API

The Shares API is part of Shield, dedicated to securely storing and retrieving user-specific secrets.

### Endpoints

#### 1. Create a Share
- **POST**: `https://shield.openfort.xyz/shares`
- **Request Headers**:
    - `Authorization`: Bearer Token
    - `x-auth-provider`: 'openfort' or 'custom'
- **Body**:
  ```json
  {
    "secret": "hjkasdhjkladshjkladhjskladhjskl",
    "user_entropy": false,
    ...
  }
  
  OR
  
  {
    "secret": "hjkasdhjkladshjkladhjskladhjskl",
    "user_entropy": true,
    "salt": "somesalt",
    "iterations": 1000,
    "length": 8,
    "digest": "SHA-256"
  }
  ```
#### 2. Get Share
- **GET**: `https://shield.openfort.xyz/shares`
- **Request Headers**:
    - `Authorization`: Bearer Token
    - `x-auth-provider`: 'openfort' or 'custom'
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
    "user_entropy": false
  }
  ```