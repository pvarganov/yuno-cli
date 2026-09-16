# Yuno API reference for yuno-cli

`docs/yuno-openapi.json` (OpenAPI 3.1.0, 119 paths, 172 operations) is the **source of truth** for
this CLI. Verify request shapes, field names, enums and path parameters against that file when
writing code or tests; treat this document as a navigation aid and resolve any discrepancy in favour
of the spec. `docs/yuno-operations.txt` is the flat `METHOD /path` listing derived from the spec and
is the authoritative coverage checklist enforced by `internal/cmd/coverage_test.go`.

Re-fetch the spec from <https://docs.y.uno/openapi.json> periodically; the coverage test fails as
soon as the spec grows an operation with no command mapped to it.

## Conventions

- **Base URLs**: `sandbox` → `https://api-sandbox.y.uno/v1`, `prod-us` → `https://api.y.uno/v1`,
  `prod-eu` → `https://api.eu.y.uno/v1`. `YUNO_API_ENDPOINT` overrides all three (used by tests).
- **Auth headers** on every request: `public-api-key`, `private-secret-key`; `X-Account-Code` and
  `X-Organization-Code` are added when the profile defines them.
- **Idempotency**: `X-Idempotency-Key` is sent on every `POST`/`PATCH`; pass `--idempotency-key` to
  pin it and make a retry safe to repeat.
- **Pagination is inconsistent across resources**: `page`/`page_size`, `limit`/`offset` and `size`
  all appear in the spec. Each command exposes the parameters its own operation declares.
- **Path parameters** become positional arguments; everything else is a flag. Large payloads come
  from `--file <path>` or `--data @-` (stdin); frequently used fields also have named flags, which
  are overlaid on top of the payload and only when explicitly set.
- **PATCH bodies** are built as `map[string]any` from `flags.Changed(...)`, so an explicit empty
  string stays distinguishable from an omitted field.

## Command to operation map

Every leaf command records the operations it calls in the cobra annotation `yuno.operations` (see
`internal/cmd/operations.go`). The table below is that annotation data, grouped by top-level
command; `internal/cmd/docs_test.go` fails if it drifts from the command tree.

### `ai-caller`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli ai-caller declined-payments` | POST /smart-support/external/payments/recover | Call the payer of a declined payment to recover it |
| `yuno-cli ai-caller recover` | POST /smart-support/external/payments | Reach out to a customer who abandoned a checkout flow |

### `banking`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli banking account close` | DELETE /banking/accounts/{account_id} | Close a bank account |
| `yuno-cli banking account create` | POST /banking/accounts | Open a bank account for a completed onboarding |
| `yuno-cli banking account get` | GET /banking/accounts/{account_id} | Retrieve one bank account with its balance |
| `yuno-cli banking account update` | PATCH /banking/accounts/{account_id} | Update a bank account |
| `yuno-cli banking entity create` | POST /banking/entities | Register a legal entity for banking connectivity |
| `yuno-cli banking entity get` | GET /banking/entities/{entity_id} | Retrieve one entity |
| `yuno-cli banking entity onboarding cancel` | POST /banking/entities/{entity_id}/onboardings/{onboarding_id}/cancel | Cancel an onboarding that has not completed |
| `yuno-cli banking entity onboarding create` | POST /banking/entities/{entity_id}/onboardings | Start the onboarding of an entity |
| `yuno-cli banking entity onboarding get` | GET /banking/entities/{entity_id}/onboardings/{onboarding_id} | Retrieve the status of an onboarding |
| `yuno-cli banking entity onboarding update` | PATCH /banking/entities/{entity_id}/onboardings/{onboarding_id} | Update an onboarding |
| `yuno-cli banking entity update` | PATCH /banking/entities/{entity_id} | Update an entity |
| `yuno-cli banking transfer cancel` | POST /banking/accounts/{account_id}/transfers/{transfer_id}/cancel | Cancel a transfer that has not settled |
| `yuno-cli banking transfer create` | POST /banking/transfers | Initiate a transfer |
| `yuno-cli banking transfer get` | GET /banking/accounts/{account_id}/transfers/{transfer_id} | Retrieve the status of a transfer |

### `campaign`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli campaign create` | POST /campaigns | Create a communications campaign |
| `yuno-cli campaign get` | GET /campaigns/{campaign_id} | Retrieve one campaign |
| `yuno-cli campaign list` | GET /campaigns | List the communications campaigns |
| `yuno-cli campaign rule create` | POST /campaigns/{campaign_id}/rules | Add targeting rules to a campaign |
| `yuno-cli campaign rule get` | GET /campaigns/{campaign_id}/rules/{rule_id} | Retrieve one rule of a campaign |
| `yuno-cli campaign rule status` | PATCH /campaigns/{campaign_id}/rules/{rule_id}/status | Enable or disable one rule of a campaign |
| `yuno-cli campaign rule update` | PATCH /campaigns/{campaign_id}/rules/{rule_id} | Change the definition of one rule of a campaign |
| `yuno-cli campaign update` | PATCH /campaigns/{campaign_id} | Change the status of a campaign |

### `checkout`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli checkout payment-method enroll` | POST /customers/sessions/{customer_session}/payment-methods | Enroll a payment method in a customer session |
| `yuno-cli checkout payment-method get` | GET /payment-methods/{payment_method_id} | Retrieve one payment method by id |
| `yuno-cli checkout payment-method list` | GET /checkout/customers/sessions/{customer_session}/payment-methods | List the payment methods a customer session can enroll |
| `yuno-cli checkout payment-method unenroll` | POST /customers/payment-methods/{payment_method_id}/unenroll | Unenroll a payment method of a customer |

### `checkout-builder`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli checkout-builder create` | POST /checkouts | Create an empty checkout configuration |
| `yuno-cli checkout-builder get` | GET /checkouts/{checkout_code} | Retrieve one checkout configuration |
| `yuno-cli checkout-builder list` | GET /checkouts | List the checkouts of the account |
| `yuno-cli checkout-builder publish` | PUT /checkouts/{checkout_code} | Write the configuration and the styling of a checkout |
| `yuno-cli checkout-builder update` | PATCH /checkouts/{checkout_code} | Rename a checkout or move it through its lifecycle |

### `checkout-session`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli checkout-session create` | POST /checkout/sessions | Open a checkout session |
| `yuno-cli checkout-session get` | GET /checkout/sessions/{checkout_session} | Retrieve one checkout session |
| `yuno-cli checkout-session payment-methods` | GET /checkout/sessions/{checkout_session}/payment-methods | List the payment methods available for a checkout session |
| `yuno-cli checkout-session update` | PATCH /checkout/sessions/{checkout_session} | Update a checkout session that has not been used yet |

### `connection`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli connection catalog` | GET /connections/catalog/{provider_id} | Show what a provider supports and which parameters it needs |
| `yuno-cli connection create` | POST /connections | Create a provider connection |
| `yuno-cli connection get` | GET /connections/{connection_id} | Retrieve one provider connection |

### `conversion-rate`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli conversion-rate get` | POST /currency-conversion | Quote the conversion of an amount into the cardholder currency |

### `customer`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli customer create` | POST /customers | Create a customer |
| `yuno-cli customer delete` | DELETE /customers/{customer_id} | Delete a customer |
| `yuno-cli customer get` | GET /customers/{customer_id} | Retrieve one customer |
| `yuno-cli customer get-by-merchant-id` | GET /customers | Retrieve a customer by your own customer id |
| `yuno-cli customer list` | GET /customers | Look a customer up by your own customer id |
| `yuno-cli customer session create` | POST /customers/sessions | Open an enrollment session for a customer |
| `yuno-cli customer update` | PATCH /customers/{customer_id} | Update a customer |

### `dry-run`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli dry-run provider-event` | POST /dry-run/provider-events | Register a simulated provider event |

### `installment-plan`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli installment-plan create` | POST /installments-plans | Create an installments plan |
| `yuno-cli installment-plan delete` | DELETE /installments-plans/{code} | Delete an installments plan |
| `yuno-cli installment-plan get` | GET /installments-plans/{code} | Retrieve one installments plan |
| `yuno-cli installment-plan list` | GET /installments-plans | List the installments plans of an account |
| `yuno-cli installment-plan update` | PATCH /installments-plans/{code} | Update an installments plan |

### `network-token`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli network-token cryptogram` | POST /network-tokens/cryptograms | Generate a cryptogram for the network token of a vaulted card |

### `org`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli org account create` | POST /organizations/account-groups/{group_id}/accounts | Create an account inside an account group |
| `yuno-cli org account delete` | DELETE /organizations/accounts/{account_id} | Delete an account |
| `yuno-cli org account get` | GET /organizations/accounts/{account_id} | Retrieve one account |
| `yuno-cli org account list` | GET /organizations/accounts | List the accounts of the organization |
| `yuno-cli org account update` | PATCH /organizations/accounts/{account_id} | Update an account |
| `yuno-cli org account-group accounts` | GET /organizations/account-groups/{group_id}/accounts | List the accounts of one account group |
| `yuno-cli org account-group add-account` | POST /organizations/account-groups/{group_id}/accounts | Create an account inside this account group |
| `yuno-cli org account-group create` | POST /organizations/account-groups | Create an account group |
| `yuno-cli org account-group delete` | DELETE /organizations/account-groups/{group_id} | Delete an account group |
| `yuno-cli org account-group find-by-merchant-id` | GET /organizations/account-groups | Find the account groups of one merchant |
| `yuno-cli org account-group get` | GET /organizations/account-groups/{group_id} | Retrieve one account group |
| `yuno-cli org account-group list` | GET /organizations/account-groups | List the account groups of the organization |
| `yuno-cli org account-group update` | PATCH /organizations/account-groups/{group_id} | Update an account group |
| `yuno-cli org authenticate` | POST /organizations/authenticate | Issue a whitelabel access token for one user |
| `yuno-cli org permissions-catalog` | GET /organizations/permissions-catalog | List every permission a role may grant |
| `yuno-cli org role create` | POST /organizations/roles | Create a role |
| `yuno-cli org role delete` | DELETE /organizations/roles/{role_id} | Delete a role |
| `yuno-cli org role list` | GET /organizations/roles | List the roles of the organization |
| `yuno-cli org role update` | PATCH /organizations/roles/{role_id} | Update a role |
| `yuno-cli org user account-group-permission delete` | DELETE /organizations/users/{user_id}/account-group-permissions/{account_group_id} | Revoke the permission of a user on one resource |
| `yuno-cli org user account-group-permission list` | GET /organizations/users/{user_id}/account-group-permissions | List the permissions of a user |
| `yuno-cli org user account-group-permission set` | PUT /organizations/users/{user_id}/account-group-permissions | Replace every permission of a user |
| `yuno-cli org user account-group-permission update` | PATCH /organizations/users/{user_id}/account-group-permissions | Add to the permissions of a user |
| `yuno-cli org user account-permission delete` | DELETE /organizations/users/{user_id}/account-permissions/{account_id} | Revoke the permission of a user on one resource |
| `yuno-cli org user account-permission list` | GET /organizations/users/{user_id}/account-permissions | List the permissions of a user |
| `yuno-cli org user account-permission set` | PUT /organizations/users/{user_id}/account-permissions | Replace every permission of a user |
| `yuno-cli org user account-permission update` | PATCH /organizations/users/{user_id}/account-permissions | Add to the permissions of a user |
| `yuno-cli org user create` | POST /organizations/users | Invite a user into the organization |
| `yuno-cli org user delete` | DELETE /organizations/users/{user_id} | Remove a user from the organization |
| `yuno-cli org user find-by-email` | GET /organizations/users | Find the user with this email |
| `yuno-cli org user get` | GET /organizations/users/{user_id} | Retrieve one user |
| `yuno-cli org user list` | GET /organizations/users | List the users of the organization |
| `yuno-cli org user update` | PATCH /organizations/users/{user_id} | Update a user |

### `payment`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli payment cancel` | POST /payments/{payment_id}/transactions/{transaction_id}/cancel | Cancel one transaction of a payment |
| `yuno-cli payment cancel-or-refund` | POST /payments/{payment_id}/cancel-or-refund<br>POST /payments/{payment_id}/transactions/{transaction_id}/cancel-or-refund | Let Yuno choose between cancelling and refunding |
| `yuno-cli payment capture` | POST /payments/{payment_id}/transactions/{transaction_id}/capture | Capture an authorized transaction of a payment |
| `yuno-cli payment create` | POST /payments | Create a payment |
| `yuno-cli payment dispute create` | POST /payments/{payment_id}/transactions/{transaction_id}/dispute | Submit the evidence of a disputed transaction |
| `yuno-cli payment dispute update` | PATCH /payments/{payment_id}/transactions/{transaction_id}/dispute | Replace the evidence of a disputed transaction |
| `yuno-cli payment fulfillments` | POST /payments/{payment_id}/fulfillments | Report the fulfillment status of a payment |
| `yuno-cli payment get` | GET /payments/{payment_id} | Retrieve one payment |
| `yuno-cli payment get-by-order-id` | GET /payments | Retrieve the payments of a merchant order |
| `yuno-cli payment issuers` | GET /issuers | List the banks available for a payment method |
| `yuno-cli payment list` | GET /payments | List the payments of a merchant order |
| `yuno-cli payment refund` | POST /payments/{payment_id}/transactions/{transaction_id}/refund | Refund one transaction of a payment |

### `payment-link`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli payment-link cancel` | POST /payment-links/{code}/cancel | Cancel a payment link so it can no longer be paid |
| `yuno-cli payment-link create` | POST /payment-links | Create a payment link |
| `yuno-cli payment-link get` | GET /payment-links/{code} | Retrieve one payment link |

### `payment-method`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli payment-method account-updater` | POST /payment-methods/account-updater | Register stored cards for the card account updater |
| `yuno-cli payment-method enroll` | POST /customers/{customer_id}/payment-methods | Enroll a payment method for a customer through the direct workflow |
| `yuno-cli payment-method get` | GET /customers/{customer_id}/payment-methods/{payment_method_id} | Retrieve one payment method enrolled for a customer |
| `yuno-cli payment-method list` | GET /customers/{customer_id}/payment-methods | List the payment methods enrolled for a customer |
| `yuno-cli payment-method unenroll` | POST /customers/{customer_id}/payment-methods/{payment_method_id}/unenroll | Unenroll a payment method of a customer |
| `yuno-cli payment-method update` | PATCH /payment-methods/{payment_method_id} | Reassign a payment method to another customer |

### `payout`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli payout create` | POST /payouts | Create a payout |
| `yuno-cli payout get` | GET /payouts/{payout_id} | Retrieve one payout |
| `yuno-cli payout list` | GET /payouts | List the payouts created under a merchant reference |

### `pci-proxy`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli pci-proxy destination create` | POST /pci-proxy/destinations | Register a destination on the allowlist |
| `yuno-cli pci-proxy destination delete` | DELETE /pci-proxy/destinations/{destination_id} | Remove a destination from the allowlist |
| `yuno-cli pci-proxy destination disable` | POST /pci-proxy/destinations/{destination_id}/disable | Disable a destination |
| `yuno-cli pci-proxy destination enable` | POST /pci-proxy/destinations/{destination_id}/enable | Enable a destination |
| `yuno-cli pci-proxy destination list` | GET /pci-proxy/destinations | List the allowlisted destinations |
| `yuno-cli pci-proxy forward` | POST /pci-proxy/forward | Forward a request to a destination with card data injected |

### `plan`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli plan create` | POST /subscriptions/plans | Create a subscription plan |
| `yuno-cli plan get` | GET /subscriptions/plans/{plan_id} | Retrieve one subscription plan |
| `yuno-cli plan list` | GET /subscriptions/plans | List the subscription plans of an account |
| `yuno-cli plan status` | POST /subscriptions/plans/{plan_id}/status | Change the status of a plan, which cancels it |

### `pre-debit`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli pre-debit create` | POST /predebit-notify | Create a pre-debit notification |
| `yuno-cli pre-debit get` | GET /predebit-notify/{notification_id} | Retrieve one pre-debit notification |
| `yuno-cli pre-debit list` | GET /predebit-notify | Look a pre-debit notification up by merchant reference |

### `recipient`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli recipient create` | POST /recipients | Create a marketplace recipient |
| `yuno-cli recipient delete` | DELETE /recipients/{recipient_id} | Delete a marketplace recipient |
| `yuno-cli recipient get` | GET /recipients/{recipient_id} | Retrieve one marketplace recipient |
| `yuno-cli recipient list` | GET /recipients | List the marketplace recipients |
| `yuno-cli recipient onboarding block` | POST /recipients/{recipient_id}/onboardings/{onboarding_id}/block | Block an onboarded recipient |
| `yuno-cli recipient onboarding cancel` | POST /recipients/{recipient_id}/onboardings/{onboarding_id}/cancel | Cancel an onboarding |
| `yuno-cli recipient onboarding continue` | POST /recipients/{recipient_id}/onboardings/{onboarding_id}/continue | Continue an onboarding that is waiting for more data |
| `yuno-cli recipient onboarding create` | POST /recipients/{recipient_id}/onboardings | Start an onboarding for a recipient |
| `yuno-cli recipient onboarding get` | GET /recipients/{recipient_id}/onboardings/{onboarding_id} | Retrieve one onboarding of a recipient |
| `yuno-cli recipient onboarding transfers` | GET /onboardings/{onboarding_id}/transfers | List the transfers of one onboarding |
| `yuno-cli recipient onboarding unblock` | POST /recipients/{recipient_id}/onboardings/{onboarding_id}/unblock | Unblock a blocked recipient |
| `yuno-cli recipient onboarding update` | PATCH /recipients/{recipient_id}/onboardings/{onboarding_id} | Update an onboarding of a recipient |
| `yuno-cli recipient transfer get` | GET /transfers/{transfer_id} | Retrieve one onboarding transfer |
| `yuno-cli recipient transfer list` | GET /recipients/{recipient_id}/transfers | List the onboarding transfers of one recipient |
| `yuno-cli recipient transfer request` | POST /recipients/{recipient_id}/onboardings/{onboarding_id}/transfer | Transfer a recipient onto another onboarding |
| `yuno-cli recipient transfer reversal` | POST /payments/{payment_id}/transactions/{transaction_id}/split-marketplace/transfer-reversal | Reverse the split marketplace transfer of a payment transaction |
| `yuno-cli recipient transfer reverse` | POST /recipients/onboardings/reverse-transfer/{transfer_id} | Reverse an onboarding transfer |
| `yuno-cli recipient update` | PATCH /recipients/{recipient_id} | Update a marketplace recipient |

### `report`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli report create` | POST /reports | Schedule a report run |
| `yuno-cli report download` | GET /reports/{report_id}/download | Retrieve the download link of a report, or the file behind it |
| `yuno-cli report get` | GET /reports/{report_id} | Retrieve one report run |
| `yuno-cli report list` | GET /reports/list | List the report runs |

### `reporting`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli reporting transactions` | POST /reporting/transactions | Ingest off-Yuno transactions |

### `routing`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli routing create` | POST /routing | Create a routing rule |
| `yuno-cli routing get` | GET /routing/{routing_id} | Retrieve one routing rule |
| `yuno-cli routing list` | GET /routing | List the routing rules of an account |
| `yuno-cli routing recommend` | POST /routing/recommendations | Ask Yuno which candidate provider to route a payment to |
| `yuno-cli routing update` | PATCH /routing/{routing_id} | Update a routing rule |

### `seller`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli seller create` | POST /sellers | Create a seller |
| `yuno-cli seller delete` | DELETE /sellers/{merchant_seller_id} | Delete a seller |
| `yuno-cli seller get` | GET /sellers/{merchant_seller_id} | Retrieve one seller |
| `yuno-cli seller update` | PUT /sellers/{merchant_seller_id} | Update a seller |

### `subscription`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli subscription cancel` | POST /subscriptions/{subscription_id}/cancel | Cancel a subscription |
| `yuno-cli subscription change-plan` | POST /subscriptions/{subscription_id}/plan | Move a subscription to another plan |
| `yuno-cli subscription create` | POST /subscriptions | Create a subscription |
| `yuno-cli subscription get` | GET /subscriptions/{subscription_id} | Retrieve one subscription |
| `yuno-cli subscription list` | GET /subscriptions | List subscriptions |
| `yuno-cli subscription pause` | POST /subscriptions/{subscription_id}/pause | Pause a subscription |
| `yuno-cli subscription payments` | GET /subscriptions/{subscription_id}/payments | List the payments of a subscription |
| `yuno-cli subscription resume` | POST /subscriptions/{subscription_id}/resume | Resume a paused subscription |
| `yuno-cli subscription retry` | POST /subscriptions/{subscription_id}/retry | Retry the last failed charge of a subscription |
| `yuno-cli subscription update` | PATCH /subscriptions/{subscription_id} | Update a subscription |

### `three-d-secure`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli three-d-secure setup` | POST /three-d-secure/setups | Create a 3-D Secure setup and collect the device fingerprints |

### `transfer`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli transfer create` | POST /split-marketplace/transfers | Create a standalone transfer to a recipient |
| `yuno-cli transfer get` | GET /split-marketplace/transfers/{transfer_id} | Retrieve one standalone transfer |
| `yuno-cli transfer reverse` | POST /split-marketplace/transfers/{transfer_id}/reverse | Reverse a standalone transfer, fully or partially |

### `webhook`

| Command | Operation | Description |
|---|---|---|
| `yuno-cli webhook create` | POST /webhooks | Register a webhook |
| `yuno-cli webhook delete` | DELETE /webhooks/{webhook_id} | Delete a webhook |
| `yuno-cli webhook get` | GET /webhooks/{webhook_id} | Retrieve one webhook |
| `yuno-cli webhook list` | GET /webhooks | List the webhooks of an account |
| `yuno-cli webhook update` | PATCH /webhooks/{webhook_id} | Update a webhook |

