# Agent Guide — Azure CI/CD Debugging

## Investigation Order

Follow the same order as Railway:
1. **Azure Portal / CLI logs** — check ACI logs and PostgreSQL metrics first
2. **CI logs** — GitHub Actions step output for surface symptoms
3. **Terraform state** — only if resource creation failed

### Check Container App logs

```bash
az containerapp logs show \
  --name "gyd-lt-<run_id>" \
  --resource-group "gyd-lt-<run_id>" \
  --follow false
```

### Check Container App status

```bash
az containerapp show \
  --name "gyd-lt-<run_id>" \
  --resource-group "gyd-lt-<run_id>" \
  --query "properties.runningStatus"
```

## Known Failure Patterns

### 1. OIDC federated credential subject mismatch

`azure/login` returns 401 or "AADSTS70021: No matching federated identity record found".

**Cause:** The `subject` claim in the federated credential doesn't match the GitHub Actions context. For `workflow_dispatch`, the subject is `repo:<owner>/<repo>:ref:refs/heads/<branch>` where the branch is whichever branch was selected in the UI, not always `develop`.

**Fix:** Use a GitHub Environment (`load-test`) and set the federated credential subject to `repo:<owner>/<repo>:environment:load-test`. Add `environment: load-test` to the job in the workflow.

### 2. ACR name already exists (global uniqueness)

`az acr create` fails with "The registry DNS name ... is already taken".

**Cause:** GitHub run IDs are unique within a repo but not globally. Extremely unlikely with the `gydlt<run_id>` naming scheme, but possible if a prior run's teardown failed before ACR deletion.

**Fix:** Check `az acr show --name gydlt<run_id>` — if it exists from a prior run, delete it manually: `az acr delete --name gydlt<run_id> --resource-group gyd-lt-<run_id> --yes`.

### 3. Container App fails to pull image — registry credentials

Container App reports `Failed to pull image` with authentication error.

**Cause:** ACR admin credentials are used for image pull. If admin is not enabled on ACR, `az acr credential show` returns an error.

**Fix:** Confirm `admin_enabled = true` in `infra/azure/main.tf`. Terraform must be applied before running `az acr credential show`.

### 4. PostgreSQL connection refused from ACI

App starts but reports `dial tcp: connect: connection refused` or SSL handshake error.

**Cause:** Either the firewall rule hasn't propagated yet (takes ~30 s after creation), or the DATABASE_URL uses `sslmode=disable` which Azure PostgreSQL rejects.

**Fix:** Ensure `sslmode=require` in the connection string. The firewall rule `0.0.0.0–255.255.255.255` covers all IPs including ACI's dynamic public IP.

### 5. Teardown fails — resource group not found

`az group delete` returns 404 after `terraform destroy` already deleted the RG.

**Cause:** Terraform destroy deletes the resource group. The subsequent `az group delete` backstop then gets a 404.

**Fix:** This is expected. The `|| true` at the end of `az group delete` suppresses the error. No action needed.

### 6. Container App cold start causes health check timeout

First request after scale-to-zero takes 5–15 s (container startup + `gatheryourdeals init`). The health check step retries every 10 s for up to 300 s, which is sufficient. If it still times out:

**Fix:** Check Container App logs for startup errors (DB connection, missing env vars). The most common cause is a failed PostgreSQL connection on first init.
