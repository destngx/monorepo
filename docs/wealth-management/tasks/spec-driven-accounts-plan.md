# Spec-Driven Development Plan for Accounts Service API

## Overview
This plan outlines the spec-driven development for implementing the accounts service API with endpoints: GetAccounts, CreateAccount, and DeleteAccount in the wealth-management-engine (Go backend).

## Structured Breakdown

### Domain Layer
- **Account Entity**: Struct with fields ID (string), Name (string), Type (string), Balance (float64), CreatedAt (time.Time), UpdatedAt (time.Time)
- **Business Rules**: Validation for account creation (e.g., name required, balance >=0), deletion checks (e.g., account exists)

### Port Layer
- **DatabaseService Interface**: Methods for GetAccounts() ([]Account, error), CreateAccount(Account) (Account, error), DeleteAccount(id string) error
- **Other Ports**: If needed, e.g., for logging or external services

### Service Layer
- **AccountsService**: Implements business logic, calls port methods, handles errors and validations

### Adapter Layer
- **Fiber Handler**: HTTP endpoints /api/accounts (GET, POST, DELETE)
- **Database Adapter**: Implementation of DatabaseService (e.g., Google Sheets adapter)

### Tests
- **Unit Tests**: For service layer
- **Integration Tests**: For handlers and adapters
- **BDD Scenarios**: Using Gherkin syntax

## Dependency Graph
```
Fiber Handler -> AccountsService -> DatabaseService (Port)
Database Adapter implements DatabaseService
```

## BDD Test Cases

### Feature: Accounts Management

#### Scenario: Retrieve all accounts when none exist
Given no accounts exist in the system
When I send a GET request to /api/accounts
Then I should receive a 200 OK response with an empty array

#### Scenario: Retrieve all accounts when some exist
Given accounts exist in the system
When I send a GET request to /api/accounts
Then I should receive a 200 OK response with the list of accounts

#### Scenario: Create a new account successfully
Given I have valid account details (name, type, balance)
When I send a POST request to /api/accounts with the details
Then I should receive a 201 Created response with the new account data

#### Scenario: Fail to create account with invalid data
Given I have invalid account details (e.g., empty name)
When I send a POST request to /api/accounts with the details
Then I should receive a 400 Bad Request response

#### Scenario: Delete an existing account
Given an account with ID exists
When I send a DELETE request to /api/accounts/{id}
Then I should receive a 204 No Content response and the account is removed

#### Scenario: Fail to delete non-existing account
Given an account with ID does not exist
When I send a DELETE request to /api/accounts/{id}
Then I should receive a 404 Not Found response

## Implementation Plan (Component-by-Component)

### Phase 1: Domain and Port Definition
1. Update domain/wealth_sheet_data.go to include Account struct if not already present (it is).
2. Create port/accounts.go with DatabaseService interface including the three methods.

### Phase 2: Service Implementation
1. Create service/accounts_service.go implementing AccountsService with business logic for the three operations.
2. Add validations and error handling.

### Phase 3: Adapter Implementation
1. Update adapter/fiber/db_handler.go to add CreateAccount and DeleteAccount handlers.
2. Implement or update database adapter (e.g., adapter/db/google_sheets.go) to support the new methods.

### Phase 4: Routing and Wiring
1. Update main.go to register the new routes for POST and DELETE /api/accounts.
2. Ensure dependency injection wires service to handlers.

### Phase 5: Testing
1. Write unit tests for AccountsService.
2. Write integration tests for handlers.
3. Run BDD tests using the scenarios above.

### Phase 6: Validation and Deployment
1. Run all tests.
2. Update documentation if needed.
3. Deploy and verify endpoints.

## Notes
- Build upon existing GetAccounts implementation.
- Follow Hexagonal architecture strictly.
- Use existing patterns from the codebase.
