# Agent Guide — Azure CI/CD Debugging

## Investigation Order

Follow the same order as Railway:
1. **Azure Portal / CLI logs** — check ACI logs and PostgreSQL metrics first
2. **CI logs** — GitHub Actions step output for surface symptoms
3. **Terraform state** — only if resource creation failed

### Check ACI logs

```bash
az container logs \
  --resource-group "gyd-lt-<run_id>" \
  --name "gyd-lt-<run_id>"
```

### Check ACI status

```bash
az container show \
  --resource-group "gyd-lt-<run_id>" \
  --name "gyd-lt-<run_id>" \
  --query "containers[0].instanceView.currentState"
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

### 3. ACI fails to pull image — registry credentials

ACI reports `Failed to pull image` with authentication error.

**Cause:** ACR admin credentials are used for ACI image pull. If admin is not enabled on ACR, `az acr credential show` returns an error.

**Fix:** Confirm `admin_enabled = true` in `infra/azure/main.tf`. Terraform must be applied before running `az acr credential show`.

### 4. PostgreSQL connection refused from ACI

App starts but reports `dial tcp: connect: connection refused` or SSL handshake error.

**Cause:** Either the firewall rule hasn't propagated yet (takes ~30 s after creation), or the DATABASE_URL uses `sslmode=disable` which Azure PostgreSQL rejects.

**Fix:** Ensure `sslmode=require` in the connection string. The firewall rule `0.0.0.0–255.255.255.255` covers all IPs including ACI's dynamic public IP.

### 5. Teardown fails — resource group not found

`az group delete` returns 404 after `terraform destroy` already deleted the RG.

**Cause:** Terraform destroy deletes the resource group. The subsequent `az group delete` backstop then gets a 404.

**Fix:** This is expected. The `|| true` at the end of `az group delete` suppresses the error. No action needed.

### 6. ACI DNS label already in use

`az container create` fails with "DNS name label not available".

**Cause:** A prior run's ACI with the same label is still being deleted (Azure DNS propagation lag), or teardown failed.

**Fix:** Wait 1–2 minutes and re-trigger, or manually delete `az container delete --resource-group gyd-lt-<run_id> --name gyd-lt-<run_id> --yes`.
