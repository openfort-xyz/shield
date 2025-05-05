# Shield

![Openfort Protocol][banner-image]

<div align="center">
  <h4>
    <a href="https://www.openfort.io/">
      Website
    </a>
    <span> | </span>
    <a href="https://www.openfort.io/docs">
      Documentation
    </a>
    <span> | </span>
    <a href="https://www.openfort.io/docs/reference/api/authentication">
      API Docs
    </a>
  </h4>
</div>

[banner-image]: https://blog-cms.openfort.xyz/uploads/Shield_85938c811e.png



[![Go Report Card](https://goreportcard.com/badge/go.openfort.xyz/shield)](https://goreportcard.com/report/go.openfort.xyz/shield)
## Overview
Shield by Openfort is a secure service dedicated to the protection of sensitive data. It ensures the confidentiality and security of data management by offering a robust framework for storing secrets and encryption parameters.

---
## **Understanding Shares, Projects, and Encryption**
#### **1. Shares**

A **share** is a crucial part of a user's private key. Shares can be stored in different ways based on their entropy:

- **Plain (Entropy: None):** The share is stored directly without encryption.
- **Encrypted by User (Entropy: User):** The share is externaly encrypted and its parameters can be stored:
  - **Salt:** A string used to introduce randomness into the encryption process.
  - **Iterations:** The number of iterations used in the encryption algorithm.
  - **Length:** The length of the resulting encrypted data.
  - **Digest:** The hashing algorithm used (e.g., SHA-256).
- **Encrypted by Project (Entropy: Project):** The share is encrypted using a project-wide encryption key. This key is split into two parts:
  - **Part 1:** Stored securely in the database.
  - **Part 2 (Encryption Part):** Provided to the client via the API when a project is created or when the endpoint to generate it is called.

#### **2. Projects**

A **project** serves as a container for a group of users and its shares and authentication methids. Projects are identified by an `API Key` and secured by an `API Secret`. The project handles encryption in a consistent manner for all its shares:

- **Project Encryption Key:** Projects can generate an encryption key in two ways:
  - **During Creation:** Using the `GenerateEncryptionKey` field in the `CreateProject` request.
  - **After Creation:** By calling the `/project/encryption-key` endpoint. This endpoint not only generates an encryption key but also encrypts all stored shares with "None" entropy.

#### **3. Handling Encryption with Shares**

**Project Entropy:** When a project encrypts shares, the following happens:

- A random encryption key is generated and split into two parts:
  - **Part 1:** Stored in the database.
  - **Part 2 (Encryption Part):** Provided through the API and required for any operation involving the share (except deletion).

**Providing the Encryption Part:** There are two ways to provide this encryption part when interacting with shares:

1. **Direct Provision:**
  - **Register/Update Share:** Include the `encryption_part` field in the request body.
  - **Get Share:** Use the `X-Encryption-Part` header.

2. **Using an Encryption Session:**
  - **Session Creation:** Call the `/project/encryption-session` endpoint with the `encryption_part` to create a session. This session is one-time use and returns a `session_id`.
  - **Register/Update Share:** Use the `encryption_session` field in the request body.
  - **Get Share:** Use the `X-Encryption-Session` header.

#### **4. User Authentication and Providers**

Users are automatically associated with a project based on the provided API key. To authenticate users (using access tokens), the project must register a provider. There are two types of providers:

1. **Openfort Provider:**
  - The project integrates with Openfort to validate user credentials.
  - When using this provider:
    - Specify `X-Auth-Provider: openfort` in the request.
    - If Openfort authentication is using Third Party provide `X-Openfort-Provider` and `X-Openfort-Token-Type` headers for user authentication details.

2. **Custom Provider:**
  - The project provides OIDC-compatible information, such as a JWK URL or a PEM certificate and key type.
  - When using this provider:
    - Specify `X-Auth-Provider: custom` in the request.

**Important Notes:**
- The `X-Auth-Provider` header is mandatory for the Shares API to specify which authentication method is being used.
- For Openfort, `X-Openfort-Provider` and `X-Openfort-Token-Type` are required headers to detail the specific authentication context.

## Endpoints
### **1. Share API Endpoints**

#### **1.1 Register Share**

- **Endpoint:** `POST /shares`
- **Request:**
  - **Type:** `RegisterShareRequest`
    - Mandatory header `Authorization` with access token and `X-API-Key` with project's api key
    - Mandatory header `X-Auth-Provider` and optional `X-Openfort-Provider` and `X-Openfort-Token-Type` for user authentication
    - Optional headers `X-Encryption-Part` and `X-Encryption-Session` to specify encryption details.
  - **Example:**
    ```json
    {
      "secret": "some_secret_value",
      "entropy": "user",
      "salt": "some_salt_value",
      "iterations": 1000,
      "length": 256,
      "digest": "sha256",
      "encryption_part": "part_value",
      "encryption_session": "session_value"
    }
    ```
- **Response:**
  - **Success:** HTTP `201 Created` with no body content.
  - **Failure:**
    - `400 Bad Request` if the request body is invalid.
    - `500 Internal Server Error` for any server-side issues.

- **How it Works:**
  - The client sends a `RegisterShareRequest` JSON payload.
  - The handler reads and validates the request data.
  - If valid, the handler registers the share using the `ShareApplication` service and returns `201 Created`.
    This endpoint can also be called with API Key, API Secret, and an extra header `X-User-ID` to register a share in name of a user.

#### **1.2 Update Share**

- **Endpoint:** `PUT /shares`
- **Request:**
  - Mandatory header `Authorization` with access token and `X-API-Key` with project's api key
  - Mandatory header `X-Auth-Provider` and optional `X-Openfort-Provider` and `X-Openfort-Token-Type` for user authentication
  - Optional headers `X-Encryption-Part` and `X-Encryption-Session` to specify encryption details.
  - **Type:** `UpdateShareRequest`
  - **Example:**
    ```json
    {
      "secret": "updated_secret_value",
      "entropy": "project",
      "salt": "updated_salt_value",
      "iterations": 2000,
      "length": 512,
      "digest": "sha512",
      "encryption_part": "updated_part_value",
      "encryption_session": "updated_session_value"
    }
    ```
- **Response:**
  - **Type:** `UpdateShareResponse`
  - **Example:**
    ```json
    {
      "secret": "updated_secret_value",
      "entropy": "project",
      "salt": "updated_salt_value",
      "iterations": 2000,
      "length": 512,
      "digest": "sha512",
      "encryption_part": "updated_part_value",
      "encryption_session": "updated_session_value"
    }
    ```
  - **Success:** HTTP `200 OK` with the updated share details.
  - **Failure:**
    - `400 Bad Request` if the request body is invalid.
    - `500 Internal Server Error` for any server-side issues.

- **How it Works:**
  - The client sends an `UpdateShareRequest` JSON payload.
  - The handler updates the share using the provided data.
  - Upon successful update, the handler returns the updated share details.

#### **1.3 Delete Share**

- **Endpoint:** `DELETE /shares`
- **Request:**
  - No request body required.
  - Mandatory header `Authorization` with access token and `X-API-Key` with project's api key
  - Mandatory header `X-Auth-Provider` and optional `X-Openfort-Provider` and `X-Openfort-Token-Type` for user authentication
  - Optional headers `X-Encryption-Part` and `X-Encryption-Session` to specify encryption details.
- **Response:**
  - **Success:** HTTP `204 No Content` indicating the share was successfully deleted.
  - **Failure:**
    - `404 Not Found` if the share does not exist.
    - `500 Internal Server Error` for any server-side issues.

- **How it Works:**
  - The client sends a request to delete the share.
  - The handler calls the `ShareApplication` service to delete the share.
  - If the deletion is successful, it returns `204 No Content`.

#### **1.4 Get Share**

- **Endpoint:** `GET /shares`
- **Request:**
  - No request body required.
  - Mandatory header `Authorization` with access token and `X-API-Key` with project's api key
  - Mandatory header `X-Auth-Provider` and optional `X-Openfort-Provider` and `X-Openfort-Token-Type` for user authentication
  - Optional headers `X-Encryption-Part` and `X-Encryption-Session` to specify encryption details.
- **Response:**
  - **Type:** `GetShareResponse`
  - **Example:**
    ```json
    {
      "secret": "some_secret_value",
      "entropy": "user",
      "salt": "some_salt_value",
      "iterations": 1000,
      "length": 256,
      "digest": "sha256",
      "encryption_part": "part_value",
      "encryption_session": "session_value"
    }
    ```
  - **Success:** HTTP `200 OK` with the share details.
  - **Failure:**
    - `404 Not Found` if the share is not found.
    - `500 Internal Server Error` for any server-side issues.

- **How it Works:**
  - The client sends a request to retrieve share details.
  - The handler fetches and returns the share details in the response.

### **2. Project API Endpoints**

#### **2.1 Create Project**

- **Endpoint:** `POST /register`
- **Request:**
  - **Type:** `CreateProjectRequest`
  - **Example:**
    ```json
    {
      "name": "My Project",
      "generate_encryption_key": true
    }
    ```
- **Response:**
  - **Type:** `CreateProjectResponse`
  - **Example:**
    ```json
    {
      "id": "project_id",
      "name": "My Project",
      "api_key": "generated_api_key",
      "api_secret": "generated_api_secret",
      "encryption_part": "generated_encryption_part"
    }
    ```
  - **Success:** HTTP `201 Created` with the project details.
  - **Failure:**
    - `400 Bad Request` if the request body is invalid.
    - `500 Internal Server Error` for any server-side issues.

- **How it Works:**
  - The client sends a `CreateProjectRequest` JSON payload.
  - The handler processes the request to create a new project.
  - The project details, including API keys and optionally an encryption part, are returned in the response.

#### **2.2 Get Project**

- **Endpoint:** `GET /project`
- **Request:**
  - No request body required.
  - Mandatory headers `X-API-Key` with project's api key and `X-API-Secret` with project's api secret
- **Response:**
  - **Type:** `GetProjectResponse`
  - **Example:**
    ```json
    {
      "id": "project_id",
      "name": "My Project"
    }
    ```
  - **Success:** HTTP `200 OK` with the project details.
  - **Failure:**
    - `404 Not Found` if the project is not found.
    - `500 Internal Server Error` for any server-side issues.

- **How it Works:**
  - The client sends a request to retrieve the project details.
  - The handler fetches and returns the project information in the response.

#### **2.3 Add Providers**

- **Endpoint:** `POST /project/providers`
- **Request:**
  - Mandatory headers `X-API-Key` with project's api key and `X-API-Secret` with project's api secret
  - **Type:** `AddProvidersRequest`
  - **Example:**
    ```json
    {
      "providers": {
        "openfort": {
          "publishable_key": "openfort_publishable_key"
        },
        "custom": {
          "jwk": "custom_jwk",
          "pem": "custom_pem",
          "key_type": "rsa"
        }
      }
    }
    ```
- **Response:**
  - **Type:** `AddProvidersResponse`
  - **Example:**
    ```json
    {
      "providers": [
        {
          "provider_id": "openfort_provider_id",
          "type": "openfort"
        },
        {
          "provider_id": "custom_provider_id",
          "type": "custom"
        }
      ]
    }
    ```
  - **Success:** HTTP `200 OK` with the list of added providers.
  - **Failure:**
    - `400 Bad Request` if the request body is invalid.
    - `500 Internal Server Error` for any server-side issues.

- **How it Works:**
  - The client sends an `AddProvidersRequest` JSON payload.
  - The handler processes the request to add providers to the project.
  - The response includes details of the added providers.

#### **2.4 Get Providers**

- **Endpoint:** `GET /project/providers`
- **Request:**
  - No request body required.
  - Mandatory headers `X-API-Key` with project's api key and `X-API-Secret` with project's api secret
- **Response:**
  - **Type:** `GetProvidersResponse`
  - **Example:**
    ```json
    {
      "providers": [
        {
          "provider_id": "openfort_provider_id",
          "type": "openfort"
        },
        {
          "provider_id": "custom_provider_id",
          "type": "custom"
        }
      ]
    }
    ```
  - **Success:** HTTP `200 OK` with the list of providers.
  - **Failure:**
    - `500 Internal Server Error` for any server-side issues.

- **How it Works:**
  - The client sends a request to retrieve all providers associated with the project.
  - The handler fetches and returns the list of providers in the response.

#### **2.5 Get Provider**

- **Endpoint:** `GET /project/providers/{provider}`
- **Request:**
  - No request body required.
  - Mandatory headers `X-API-Key` with project's api key and `X-API-Secret` with project's api secret
- **Response:**
  - **Type:** `GetProviderResponse`
  - **Example:**
    ```json
    {
      "provider_id": "custom_provider_id",
      "type": "custom",
      "jwk": "custom_jwk",
      "pem": "custom_pem",
      "key_type": "rsa"
    }
    ```
  - **Success:** HTTP `200 OK` with the provider details.
  - **Failure:**
    - `404 Not Found` if the provider is not found.
    - `500 Internal Server Error` for any server-side issues.

- **How it Works:**
  - The client sends a request to retrieve details of a specific provider by its ID.
  - The handler fetches and returns the provider details.

#### **2.6 Update Provider**

- **Endpoint:** `PUT /project/providers/{provider}`
- **Request:**
  - Mandatory headers `X-API-Key` with project's api key and `X-API-Secret` with project's api secret
  - **Type:** `UpdateProviderRequest`
  - **Example:**
    ```json
    {
      "publishable_key": "new_publishable_key",
      "jwk": "new_jwk",
      "pem": "new_pem",
      "key_type": "ecdsa"
    }
    ```
- **Response:**
  - **Success:** HTTP `200 OK` indicating the provider was updated successfully.
  - **Failure:**
    - `400 Bad Request` if the request body is invalid.
    - `500 Internal Server Error` for any server-side issues.

- **How it Works:**
  - The client sends an `UpdateProviderRequest` JSON payload with updated details for the provider.
  - The handler processes the request to update the provider’s configuration.
  - If successful, the handler returns a `200 OK`.

#### **2.7 Delete Provider**

- **Endpoint:** `DELETE /project/providers/{provider}`
- **Request:**
  - No request body required.
  - Mandatory headers `X-API-Key` with project's api key and `X-API-Secret` with project's api secret
- **Response:**
  - **Success:** HTTP `200 OK` indicating the provider was successfully deleted.
  - **Failure:**
    - `404 Not Found` if the provider does not exist.
    - `500 Internal Server Error` for any server-side issues.

- **How it Works:**
  - The client sends a request to delete a provider by its ID.
  - The handler calls the `ProjectApplication` service to remove the provider from the project.
  - Upon successful deletion, it returns `200 OK`.

#### **2.8 Encrypt Project Shares**

- **Endpoint:** `POST /project/encrypt`
- **Request:**
  - Mandatory headers `X-API-Key` with project's api key and `X-API-Secret` with project's api secret
  - **Type:** `EncryptBodyRequest`
  - **Example:**
    ```json
    {
      "encryption_part": "encryption_part_value"
    }
    ```
- **Response:**
  - **Success:** HTTP `200 OK` indicating the shares were successfully encrypted.
  - **Failure:**
    - `500 Internal Server Error` for any server-side issues.

- **How it Works:**
  - The client sends an `EncryptBodyRequest` JSON payload to encrypt all project shares.
  - The handler processes the request and returns `200 OK` if encryption is successful.

#### **2.9 Register Encryption Session**

- **Endpoint:** `POST /project/encryption-session`
- **Request:**
  - Mandatory headers `X-API-Key` with project's api key and `X-API-Secret` with project's api secret
  - **Type:** `RegisterEncryptionSessionRequest`
  - **Example:**
    ```json
    {
      "encryption_part": "encryption_part_value"
    }
    ```
- **Response:**
  - **Type:** `RegisterEncryptionSessionResponse`
  - **Example:**
    ```json
    {
      "session_id": "generated_session_id"
    }
    ```
  - **Success:** HTTP `200 OK` with the session ID.
  - **Failure:**
    - `500 Internal Server Error` for any server-side issues.

- **How it Works:**
  - The client sends a `RegisterEncryptionSessionRequest` JSON payload to register a session.
  - The handler processes the request and returns the generated session ID.

#### **2.10 Register Encryption Key**

- **Endpoint:** `POST /project/encryption-key`
- **Request:**
  - No request body required.
  - Mandatory headers `X-API-Key` with project's api key and `X-API-Secret` with project's api secret
- **Response:**
  - **Type:** `RegisterEncryptionKeyResponse`
  - **Example:**
    ```json
    {
      "encryption_part": "generated_encryption_part"
    }
    ```
  - **Success:** HTTP `200 OK` with the registered encryption part.
  - **Failure:**
    - `500 Internal Server Error` for any server-side issues.

- **How it Works:**
  - The client sends a request to register a new encryption key for the project.
  - The handler processes the request and returns the generated encryption part.
