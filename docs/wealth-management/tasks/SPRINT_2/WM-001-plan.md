# Spec-Driven Development Plan for Accounts Service API

## Structured Breakdown of the Spec

### Vision
The accounts service API provides CRUD operations for managing financial accounts in the wealth management platform. It allows users to retrieve all accounts, create new accounts, and delete existing accounts. The API ensures data integrity, caching for performance, and integration with Google Sheets as the persistent store.

### Tech Stack
- **Backend**: Go with Fiber framework for HTTP handling
- **Database**: Google Sheets for data persistence
- **Caching**: Redis (Upstash) for performance
- **Architecture**: Hexagonal (Ports & Adapters)
- **Testing**: TDD/BDD with Go tests

### Components with Tier Assignments
- **Domain Tier**: `Account` struct, `AccountCreateInput` struct (business entities)
- **Port Tier**: `DatabaseService` interface (contracts for business logic)
- **Service Tier**: `dbService` (business logic implementation)
- **Adapter Tier**: `DatabaseHandler` (HTTP adapter), `google_sheets` client (infrastructure adapter)

## Dependency Graph
```
Domain (Account, AccountCreateInput)
    ↓
Port (DatabaseService interface)
    ↓
Service (dbService implementation)
    ↓
Adapter (DatabaseHandler for HTTP, google_sheets for persistence)
```

Order of Implementation:
1. Domain → 2. Port → 3. Service → 4. Adapter (Sheets) → 5. Adapter (Handler) → 6. Main (Routes)

## BDD-Style Test Cases

### GetAccounts
**Given** the accounts sheet contains valid account data  
**When** a GET request is made to `/api/accounts`  
**Then** the response should return a JSON array of account objects with status 200  
**And** the accounts should be cached for subsequent requests  

**Given** the cache is empty and sheets read fails  
**When** a GET request is made to `/api/accounts`  
**Then** the response should return an error with status 500  

### CreateAccount
**Given** valid account creation data (name, type, balance)  
**When** a POST request is made to `/api/accounts` with the data  
**Then** the account should be appended to the Google Sheet  
**And** the accounts cache should be invalidated  
**And** the response should return success with status 200  

**Given** invalid account data (missing required name)  
**When** a POST request is made to `/api/accounts`  
**Then** the response should return a validation error with status 400  

### DeleteAccount
**Given** an account exists with a specific name  
**When** a DELETE request is made to `/api/accounts/{name}`  
**Then** the account row should be marked as deleted in the sheet  
**And** the accounts cache should be invalidated  
**And** the response should return success with status 200  

**Given** an account does not exist  
**When** a DELETE request is made to `/api/accounts/{name}`  
**Then** the response should return a not found error with status 404  

## Component-by-Component Implementation Plan

### Phase 1: Domain Layer
- Add `AccountCreateInput` struct to `domain/wealth_sheet_data.go`
- Fields: Name (required), Type (required), Balance (optional), Currency (optional), Note (optional)
- Validation: Ensure Name is not empty, Type is valid enum

**Validation Checkpoint**: Compile successfully, no syntax errors

### Phase 2: Port Layer
- Add `CreateAccount(input AccountCreateInput) error` to `DatabaseService` interface in `port/db.go`
- Add `DeleteAccount(name string) error` to `DatabaseService` interface

**Validation Checkpoint**: Interface compiles, no missing methods

### Phase 3: Service Layer
- Implement `CreateAccount` in `dbService`:
  - Validate input
  - Map to sheet row format
  - Call `client.WriteToFirstEmptyRow("Accounts", accountsAnchorRange, values)`
  - Invalidate cache pattern "accounts:*"
- Implement `DeleteAccount` in `dbService`:
  - Read all accounts
  - Find row by name
  - Update row to mark as deleted (set name to "[DELETED] {name}" or clear row)
  - Invalidate cache

**Validation Checkpoint**: Unit tests pass for service methods

### Phase 4: Adapter Layer (Sheets)
- Ensure `google_sheets` client supports `WriteToFirstEmptyRow` and `UpdateRow` (already implemented for transactions)

**Validation Checkpoint**: Integration test with sheets succeeds

### Phase 5: Adapter Layer (Handler)
- Add `CreateAccount` handler in `DatabaseHandler`:
  - Parse JSON body to `AccountCreateInput`
  - Validate required fields
  - Call `service.CreateAccount(input)`
  - Return success/error
- Add `DeleteAccount` handler:
  - Extract name from URL param
  - Call `service.DeleteAccount(name)`
  - Return success/error

**Validation Checkpoint**: Handler tests pass

### Phase 6: Main Layer
- Register routes in `main.go`:
  - `api.Post("/accounts", dbHandler.CreateAccount, meta)`
  - `api.Delete("/accounts/:name", dbHandler.DeleteAccount, meta)`

**Validation Checkpoint**: Server starts without errors, routes registered

### Phase 7: Testing & Validation
- Write BDD tests in `adapter/fiber/db_handler_test.go`
- Run `bunx nx run wealth-management-engine:test`
- Perform e2e validation with live sheets if possible

**Validation Checkpoint**: All tests pass, API responds correctly
