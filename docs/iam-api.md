# IAM API Documentation

This document describes the API endpoints for the Identity and Access Management (IAM) system.

## Policy Management

### List Policies

`GET /api/v1/policies`

Returns a list of all policies.

**Response:**

```json
{
  "policies": [
    {
      "id": 1,
      "name": "AdministratorAccess",
      "type": "system",
      "document": { ... }
    }
  ]
}
```

### Create Policy

`POST /api/v1/policies`

Creates a new custom policy.

**Request:**

```json
{
  "name": "MyPolicy",
  "description": "My custom policy",
  "document": {
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Action": "compute:ListInstances",
        "Resource": "*"
      }
    ]
  }
}
```

### Get Policy

`GET /api/v1/policies/:id`

Returns details of a specific policy.

### Update Policy

`PUT /api/v1/policies/:id`

Updates a custom policy. System policies cannot be modified.

**Request:**

```json
{
  "description": "Updated description",
  "document": { ... }
}
```

### Delete Policy

`DELETE /api/v1/policies/:id`

Deletes a custom policy. System policies cannot be deleted.

## Policy Attachment

### Attach Policy to User

`POST /api/v1/users/:userId/policies/:policyId`

Attaches a policy to a user.

### Detach Policy from User

`DELETE /api/v1/users/:userId/policies/:policyId`

Detaches a policy from a user.

## Policy Document Structure

The policy document follows the AWS IAM JSON policy format.

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "OptionalStatementID",
      "Effect": "Allow" | "Deny",
      "Action": "service:action" | ["service:action", ...],
      "Resource": "resource-arn" | ["resource-arn", ...]
    }
  ]
}
```

### Evaluation Logic

1. **Default Deny**: If no policy explicitly allows an action, it is denied.
2. **Explicit Deny**: If any policy denies an action, it is denied (overrides Allow).
3. **Allow**: If a policy allows an action and no policy denies it, it is allowed.
