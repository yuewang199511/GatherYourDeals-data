# Implementation Plan for GatherYourDeals-data

**Created:** February 9, 2026  
**Status:** Planning Phase

## Overview

This document outlines the foundational work needed before Copilot can effectively generate consistent, scalable code from GitHub issues.

## File Structure to Create

```
GatherYourDeals-data/
├── .github/
│   ├── copilot-instructions.md     # Copilot-specific guidelines
│   └── workflows/
│       └── ci.yml                   # GitHub Actions CI/CD
├── docs/                            # ✅ Existing design docs
│   ├── connection_and_auth.md       # ✅ Done
│   ├── data_format.md               # ✅ Done
│   ├── api_design.md                # TODO: API specification
│   └── database_schema.md           # TODO: Database design
├── src/                             # TODO: Source code
│   └── gatheryourdeals/
│       ├── __init__.py
│       ├── api/                     # FastAPI routes
│       ├── models/                  # Database models
│       ├── auth/                    # Authentication logic
│       └── services/                # Business logic
├── tests/                           # TODO: Test files
├── migrations/                      # TODO: Database migrations
├── .gitignore                       # TODO: Python-specific
├── .pre-commit-config.yaml          # TODO: Code quality hooks
├── pyproject.toml                   # TODO: Dependencies & config
├── CONTRIBUTING.md                  # TODO: Coding standards
├── LICENSE
└── README.md                        # ✅ Done
```

---

## Phase 1: Foundation Setup (Week 1)

### Issues to Create:

- [ ] **#1: Setup project structure and configuration files**
  - Create `pyproject.toml` with dependencies
  - Create `.pre-commit-config.yaml` for code quality
  - Create `.gitignore` for Python
  - Create `CONTRIBUTING.md` with coding standards
  - Create `.github/copilot-instructions.md`

- [ ] **#2: Define API design specification**
  - Create `docs/api_design.md` based on `connection_and_auth.md`
  - List all endpoints with request/response formats
  - Define standard error response format
  - Document authentication flow

- [ ] **#3: Design database schema**
  - Create `docs/database_schema.md` based on `data_format.md`
  - Define tables (owners, access_keys, purchases, custom_fields)
  - Create initial migration file
  - Document relationships and constraints

- [ ] **#4: Initialize project structure**
  - Create `src/gatheryourdeals/` package structure
  - Create empty module files with docstrings
  - Setup `tests/` directory structure
  - Create sample test file as template

- [ ] **#5: Setup CI/CD pipeline**
  - Create `.github/workflows/ci.yml`
  - Configure automated testing (pytest)
  - Configure linting (black, flake8, mypy)
  - Configure pre-commit hooks

---

## Phase 2: Core Implementation (Week 2-3)

**Only start after Phase 1 is complete!**

### Model Layer:
- [ ] **#6: Implement database models**
  - `src/gatheryourdeals/models/owner.py`
  - `src/gatheryourdeals/models/access_key.py`
  - `src/gatheryourdeals/models/purchase.py`
  - Include SQLAlchemy/Pydantic models with type hints

### Authentication:
- [ ] **#7: Implement Ed25519 signature verification**
  - `src/gatheryourdeals/auth/crypto.py`
  - Public key validation
  - Signature verification logic
  - Unit tests with test vectors

- [ ] **#8: Implement owner registration**
  - `src/gatheryourdeals/auth/registration.py`
  - Username + public key validation
  - Owner creation logic
  - Unit tests

### API Endpoints:
- [ ] **#9: Create authentication endpoints**
  - `POST /api/v1/auth/register`
  - `POST /api/v1/auth/verify-signature`
  - `POST /api/v1/auth/create-access-key`
  - `DELETE /api/v1/auth/revoke-key/{key_id}`

- [ ] **#10: Create purchase CRUD endpoints**
  - `GET /api/v1/purchases`
  - `POST /api/v1/purchases`
  - `GET /api/v1/purchases/{id}`
  - `PUT /api/v1/purchases/{id}`
  - `DELETE /api/v1/purchases/{id}`

- [ ] **#11: Implement custom field management**
  - `POST /api/v1/metadata/fields`
  - `GET /api/v1/metadata/fields`
  - Field validation logic

---

## Phase 3: Testing & Documentation (Week 4)

- [ ] **#12: Integration tests**
  - End-to-end API testing
  - Authentication flow tests
  - Access key permission tests

- [ ] **#13: API documentation**
  - Generate OpenAPI/Swagger docs
  - Add usage examples
  - Document error codes

- [ ] **#14: Performance optimization**
  - Add database indexes
  - Query optimization
  - Caching strategy

---

## Tech Stack Decisions Needed

Before starting Phase 1, decide on:

### Backend Framework:
- [ ] **Python + FastAPI** (recommended for your use case)
  - Pros: Fast, modern, automatic API docs, type hints, async support
  - Cons: Newer ecosystem
- [ ] **Python + Flask**
  - Pros: Mature, simple, flexible
  - Cons: More boilerplate, no built-in async
- [ ] **Node.js + Express**
  - Pros: JavaScript everywhere, large ecosystem
  - Cons: Less type safety (even with TypeScript)
- [ ] **Go + Gin/Echo**
  - Pros: Performance, compiled binary, simple deployment
  - Cons: More verbose, smaller ecosystem

### Database:
- [ ] **SQLite** (for local hosting)
  - Pros: Zero config, single file, perfect for local
  - Cons: Limited concurrent writes
- [ ] **PostgreSQL** (for cloud hosting)
  - Pros: Full-featured, reliable, good for multi-user
  - Cons: Requires server setup

**Recommendation:** Support both with abstraction layer

### ORM/Database Library:
- [ ] **SQLAlchemy** (Python)
- [ ] **Prisma** (Node.js)
- [ ] **GORM** (Go)

### Testing:
- [ ] **pytest** (Python)
- [ ] **Jest** (Node.js)
- [ ] **Go testing** (Go)

---

## Coding Standards to Define

### For CONTRIBUTING.md:

1. **Code Style:**
   - Formatter: Black (Python) / Prettier (JS) / gofmt (Go)
   - Linter: Flake8 + mypy (Python) / ESLint (JS)
   - Line length: 88 (Black default) or 100
   - Use type hints everywhere

2. **Naming Conventions:**
   - Functions: `snake_case`
   - Classes: `PascalCase`
   - Constants: `UPPER_SNAKE_CASE`
   - Private: `_leading_underscore`

3. **Documentation:**
   - Docstrings for all public functions/classes
   - Type hints required
   - Example usage in docstrings

4. **Testing:**
   - Minimum 80% code coverage
   - Unit tests for all business logic
   - Integration tests for API endpoints
   - Test file naming: `test_*.py`

5. **Git Workflow:**
   - Branch naming: `feature/`, `fix/`, `docs/`
   - Always use PRs, never push to main directly
   - Reference issue in PR: "Fixes #123"
   - Squash commits when merging

6. **API Design:**
   - RESTful conventions
   - Versioned endpoints: `/api/v1/`
   - Consistent error responses
   - Use HTTP status codes correctly

---

## When to Use Copilot for Issues

### ✅ Use Copilot After Phase 1 Complete:
Once you have:
1. ✅ Project structure defined
2. ✅ Coding standards documented
3. ✅ API design specified
4. ✅ Database schema defined
5. ✅ Sample implementation as template

### ❌ Don't Use Copilot Yet:
Without foundations, Copilot will:
- Generate inconsistent code
- Use different patterns each time
- Create duplicate utilities
- Not follow your conventions

---

## Example Issue Template

Once Phase 1 is done, issues should look like:

```markdown
## Issue #9: Create Authentication Endpoints

### Description
Implement authentication endpoints according to `docs/api_design.md` and `docs/connection_and_auth.md`.

### Requirements
- [ ] POST /api/v1/auth/register - Register new owner
- [ ] POST /api/v1/auth/verify-signature - Verify owner signature
- [ ] POST /api/v1/auth/create-access-key - Create read-only access key
- [ ] DELETE /api/v1/auth/revoke-key/{key_id} - Revoke access key

### Acceptance Criteria
- Follows coding standards in CONTRIBUTING.md
- Includes type hints
- Has unit tests with >80% coverage
- Returns consistent error format
- Includes docstrings

### Files to Create/Modify
- `src/gatheryourdeals/api/auth.py`
- `tests/test_auth_endpoints.py`

### Reference
- Design: `docs/connection_and_auth.md`
- API Spec: `docs/api_design.md`
```

---

## Next Steps

1. **Tomorrow:** Review this plan
2. **Decide on tech stack** (Python + FastAPI recommended)
3. **Create Phase 1 issues** in GitHub
4. **Let Copilot help** with Phase 1 setup
5. **After Phase 1:** Copilot can implement Phase 2 with minimal adjustments

---

## Notes

- Don't rush to implementation
- Foundations = Consistent, scalable code
- Phase 1 is ~20% of time but 80% of quality impact
- Once setup is done, Copilot becomes **very powerful**

**Rest well! Review this when you're ready to continue.**
