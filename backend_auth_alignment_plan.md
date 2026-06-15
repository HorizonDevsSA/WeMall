# Backend Auth Alignment Plan

This document outlines the proposed changes to align the WeMall backend (`user-service` and `api-gateway`) with the user authentication capabilities and interface designs of the frontend Android application.

## 1. Analysis of Current Inconsistencies

### A. Google Sign-In Authentication Flow
- **Current Backend Flow**: 
  - The GraphQL gateway mutation `buyerGoogleAuth(code: String!, redirectUri: String)` and the `user-service` gRPC method `BuyerGoogleAuth(GoogleAuthRequest)` expect an OAuth **Authorization Code** (`code`).
  - The backend attempts to exchange this code with Google's OAuth endpoints and fetches user info from Google's APIs.
- **Frontend UI Flow**:
  - The Android app uses client-side Google Sign-In (Credential Manager/Google Play Services) to authenticate the user and obtain the user profile (`email`, `fullName`) and an **ID Token** directly on the device.
  - The frontend ViewModel `AuthViewModel.loginWithGoogle` expects to pass `email`, `name`, and `idToken` to the repository.
- **Alignment Requirement**: The backend needs to accept pre-authenticated user credentials (`email`, `fullName`, and `idToken`) instead of a raw Authorization Code. The backend should verify the integrity of the `idToken` using Google API Client libraries.

### B. OTP Verification Digit Length
- **Current Backend Flow**: 
  - The `user-service` generates and expects a **6-digit** OTP code (`const otpLength = 6` in `auth_service.go`).
- **Frontend UI Flow**:
  - The Android app `AuthScreen.kt` implements a **6-digit** verification code entry interface (`val otpLength = 4` with 4 separate digit entry fields).
- **Alignment Requirement**: The backend OTP length needs to be reduced from `6` to `4` to prevent mismatches during mobile login verification.

---

## 2. Proposed Backend Changes

### Phase 1: Proto & Schema Definition Updates

#### [MODIFY] [user.proto](file:///Volumes/Untitled/WeMall/proto/user/v1/user.proto)
- Update or add the request message to support token-based Google Authentication:
  ```protobuf
  message GoogleSignInRequest {
    string email     = 1;
    string full_name = 2;
    string id_token  = 3;
  }
  ```
- Expose the new RPC method under `UserService`:
  ```protobuf
  rpc BuyerGoogleSignIn(GoogleSignInRequest) returns (AuthResponse);
  ```

#### [MODIFY] [schema.graphql](file:///Volumes/Untitled/WeMall/services/api-gateway/internal/graph/schema/schema.graphql)
- Expose the corresponding mutation in the GraphQL schema:
  ```graphql
  buyerGoogleSignIn(email: String!, fullName: String!, idToken: String!): AuthPayload!
  ```

---

### Phase 2: Service Layer Implementation

#### [MODIFY] [auth_service.go](file:///Volumes/Untitled/WeMall/services/user-service/internal/service/auth_service.go)
- **OTP Length Adjustment**:
  - Modify the constant `otpLength` from `6` to `4`:
    ```go
    const (
        otpLength   = 4
        otpTTL      = 5 * time.Minute
        maxAttempts = 3
    )
    ```
- **New Google Auth Method**:
  - Implement `BuyerGoogleSignIn` using Google Identity libraries (e.g. `google.golang.org/api/idtoken`) to verify the client's `idToken` integrity:
    ```go
    func (s *AuthService) BuyerGoogleSignIn(ctx context.Context, email, fullName, idToken string) (*AuthTokens, *db.User, error) {
        // 1. Verify idToken against Google's keys
        payload, err := idtoken.Validate(ctx, idToken, s.cfg.GoogleClientID)
        if err != nil {
            return nil, nil, fmt.Errorf("invalid google id token: %w", err)
        }
        
        // 2. Cross check verified email with the request
        verifiedEmail := payload.Claims["email"].(string)
        if verifiedEmail != email {
            return nil, nil, fmt.Errorf("email mismatch")
        }

        // 3. Upsert user using the verified details
        user, err := s.q.UpsertGoogleUser(ctx, db.UpsertGoogleUserParams{
            Email:     &verifiedEmail,
            FullName:  fullName,
            GoogleID:  &payload.Subject,
        })
        if err != nil {
            return nil, nil, fmt.Errorf("upsert user: %w", err)
        }

        // 4. Generate access & refresh tokens
        tokens, err := s.issueTokens(ctx, user.ID.String(), string(user.Role))
        if err != nil {
            return nil, nil, err
        }
        return tokens, &user, nil
    }
    ```

#### [MODIFY] [user_handler.go](file:///Volumes/Untitled/WeMall/services/user-service/internal/handler/user_handler.go)
- Bind the new gRPC request to the `AuthService`:
  ```go
  func (h *UserHandler) BuyerGoogleSignIn(ctx context.Context, req *userv1.GoogleSignInRequest) (*userv1.AuthResponse, error) {
      tokens, user, err := h.authSvc.BuyerGoogleSignIn(ctx, req.Email, req.FullName, req.IdToken)
      if err != nil {
          return nil, err
      }
      return &userv1.AuthResponse{
          AccessToken:  tokens.AccessToken,
          RefreshToken: tokens.RefreshToken,
          User:         toUserProto(user),
      }, nil
  }
  ```

---

### Phase 3: API Gateway & Resolvers

#### [MODIFY] [schema.resolvers.go](file:///Volumes/Untitled/WeMall/services/api-gateway/internal/graph/schema.resolvers.go)
- Implement the resolver for the new `buyerGoogleSignIn` mutation:
  ```go
  func (r *mutationResolver) BuyerGoogleSignIn(ctx context.Context, email string, fullName string, idToken string) (*model.AuthPayload, error) {
      resp, err := r.userClient.BuyerGoogleSignIn(ctx, &userv1.GoogleSignInRequest{
          Email:    email,
          FullName: fullName,
          IdToken:  idToken,
      })
      if err != nil {
          return nil, err
      }
      return &model.AuthPayload{
          AccessToken:  resp.AccessToken,
          RefreshToken: resp.RefreshToken,
          User:         mapUser(resp.User),
      }, nil
  }
  ```

---

## 3. Verification & Execution Plan

1. **Rebuild Protos**:
   - Run `buf generate` or `make proto` from the root of the backend to generate Go/gRPC structures.
2. **Backend Unit Testing**:
   - Add unit tests verifying that 4-digit OTPs are correctly stored, mapped to SMS templates, and successfully resolved.
   - Mock token verification tests in `auth_service_test.go` to test google token workflows.
3. **Integration Testing**:
   - Update `test_nearby_and_follows.sh` or write a new script simulating the 4-digit OTP code validation.
