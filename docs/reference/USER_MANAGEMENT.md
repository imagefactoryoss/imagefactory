# User Management

This document summarizes current user management behavior in Image Factory.

## Overview

- Users are managed within tenant context.
- Role assignment is handled through tenant groups.
- Invitations are created and tracked in `user_invitations`.

---

## Key Tables

- `users`
- `user_invitations`
- `tenant_groups`
- `group_members`

---

## Key APIs

- User listing and lookup endpoints
- Invitation creation and acceptance
