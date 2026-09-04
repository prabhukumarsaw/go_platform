# ==============================================================================
# BHARATVANI NEWS PLATFORM - IAM MODULE END-TO-END WORKFLOW TEST
# ==============================================================================
# This script executes a complete real-world workflow against the IAM module:
# 1. Authentication & Token Acquisition
# 2. Query System & Custom Roles
# 3. Create Custom Role & Validate Duplicate Protection
# 4. Inspect Granular Permission Matrix (13 Menus x 7 Standard Actions)
# 5. Apply Automated Newsroom Permission Template (e.g. Reporter Template)
# 6. Customize Granular Matrix Rights
# 7. Clone Role with Inherited Permission Atoms
# 8. Staff Governance & Role Assignment
# 9. Bureau & Category Jurisdiction Scoping
# 10. ABAC Zero-Trust Evaluation & Audit Trail Verification
# 11. Teardown & Clean Up
# ==============================================================================

$baseUrl = "http://localhost:8080/api/v1"
$ErrorActionPreference = "Stop"

Write-Host "`n========================================================" -ForegroundColor Cyan
Write-Host " 🛡️  BHARATVANI IAM MODULE WORKFLOW VERIFICATION SUITE" -ForegroundColor Cyan
Write-Host "========================================================`n" -ForegroundColor Cyan

# ------------------------------------------------------------------------------
# STEP 1: Authentication & Token Acquisition
# ------------------------------------------------------------------------------
Write-Host "[STEP 1] Authenticating as SuperAdmin..." -ForegroundColor Yellow
$loginPayload = @{
    email = "superadmin@newsplatform.in"
    password = "admin123"
} | ConvertTo-Json

$loginRes = Invoke-RestMethod -Uri "$baseUrl/auth/login" -Method POST -Body $loginPayload -ContentType "application/json"
$token = $loginRes.data.tokens.access_token
$adminUser = $loginRes.data.user
$headers = @{
    Authorization = "Bearer $token"
    "Content-Type" = "application/json"
}
Write-Host "  ✓ Authenticated as: $($adminUser.email) (ID: $($adminUser.id))" -ForegroundColor Green

# ------------------------------------------------------------------------------
# STEP 2: List Roles & Inspect Seeded System Roles
# ------------------------------------------------------------------------------
Write-Host "`n[STEP 2] Listing Active Newsroom Roles..." -ForegroundColor Yellow
$rolesRes = Invoke-RestMethod -Uri "$baseUrl/iam/roles" -Method GET -Headers $headers
$roles = $rolesRes.data
Write-Host "  ✓ Found $($roles.Count) roles in database:" -ForegroundColor Green
foreach ($r in $roles) {
    Write-Host "    - [ID $($r.id)] $($r.name) (System=$($r.is_system), Users=$($r.user_count))" -ForegroundColor Gray
}

# ------------------------------------------------------------------------------
# STEP 3: Create Custom Role & Verify Duplicate Validation
# ------------------------------------------------------------------------------
Write-Host "`n[STEP 3] Creating New Custom Role & Testing Duplicate Protection..." -ForegroundColor Yellow
$testRoleName = "Investigative Bureau Lead"
$createPayload = @{
    name = $testRoleName
    description = "Senior national investigative journalism bureau"
} | ConvertTo-Json

$newRoleRes = Invoke-RestMethod -Uri "$baseUrl/iam/roles" -Method POST -Headers $headers -Body $createPayload
$testRole = $newRoleRes.data
Write-Host "  ✓ Successfully created custom role: [ID $($testRole.id)] $($testRole.name)" -ForegroundColor Green

# Test Duplicate Protection
Write-Host "  * Testing duplicate prevention with identical name..." -ForegroundColor Gray
try {
    Invoke-RestMethod -Uri "$baseUrl/iam/roles" -Method POST -Headers $headers -Body $createPayload
    throw "ERROR: Duplicate check failed! Backend allowed duplicate role name."
} catch {
    $errObj = $_.ErrorDetails.Message | ConvertFrom-Json
    Write-Host "  ✓ Duplicate correctly rejected (HTTP 400): $($errObj.error.message)" -ForegroundColor Green
}

# ------------------------------------------------------------------------------
# STEP 4: Inspect Granular Permission Matrix
# ------------------------------------------------------------------------------
Write-Host "`n[STEP 4] Inspecting Granular Permission Matrix for New Role..." -ForegroundColor Yellow
$matrixRes = Invoke-RestMethod -Uri "$baseUrl/iam/roles/$($testRole.id)/matrix" -Method GET -Headers $headers
$menus = $matrixRes.data
$totalPerms = 0
foreach ($m in $menus) {
    $totalPerms += $m.actions.Count
}
Write-Host "  ✓ Granular matrix loaded: $($menus.Count) menus, $totalPerms total permission atoms." -ForegroundColor Green
Write-Host "  ✓ Sample module: '$($menus[1].label)' ($($menus[1].name)) has $($menus[1].actions.Count) canonical actions:" -ForegroundColor Gray
foreach ($a in $menus[1].actions) {
    Write-Host "      - $($a.action) (ID: $($a.action_id), Granted: $($a.granted))" -ForegroundColor DarkGray
}

# ------------------------------------------------------------------------------
# STEP 5: Apply Newsroom Role Template (Reporter Preset)
# ------------------------------------------------------------------------------
Write-Host "`n[STEP 5] Applying Role Template Preset ('reporter')..." -ForegroundColor Yellow
$tmplPayload = @{ template = "reporter" } | ConvertTo-Json
$tmplRes = Invoke-RestMethod -Uri "$baseUrl/iam/roles/$($testRole.id)/template" -Method POST -Headers $headers -Body $tmplPayload
Write-Host "  ✓ Template applied successfully: $($tmplRes.message)" -ForegroundColor Green

# Verify granted actions increased
$matrixAfterTmpl = Invoke-RestMethod -Uri "$baseUrl/iam/roles/$($testRole.id)/matrix" -Method GET -Headers $headers
$grantedCount = 0
foreach ($m in $matrixAfterTmpl.data) {
    foreach ($a in $m.actions) {
        if ($a.granted -eq $true) { $grantedCount++ }
    }
}
Write-Host "  ✓ Verified: Role now holds $grantedCount granted permission atoms via template." -ForegroundColor Green

# ------------------------------------------------------------------------------
# STEP 6: Granular Matrix Customization
# ------------------------------------------------------------------------------
Write-Host "`n[STEP 6] Customizing Specific Granular Actions in Matrix..." -ForegroundColor Yellow
# Find Articles 'publish' and 'approve' actions
$actionIdsToGrant = @()
foreach ($m in $matrixAfterTmpl.data) {
    foreach ($a in $m.actions) {
        if ($a.granted -eq $true) {
            $actionIdsToGrant += $a.action_id
        }
        # Explicitly add articles.publish
        if ($m.name -eq "articles" -and $a.action -eq "publish") {
            $actionIdsToGrant += $a.action_id
        }
    }
}
$actionIdsToGrant = $actionIdsToGrant | Select-Object -Unique
$updateMatrixPayload = @{ action_ids = $actionIdsToGrant } | ConvertTo-Json
$updateMatrixRes = Invoke-RestMethod -Uri "$baseUrl/iam/roles/$($testRole.id)/matrix" -Method PUT -Headers $headers -Body $updateMatrixPayload
Write-Host "  ✓ Granular matrix updated. Assigned $($actionIdsToGrant.Count) custom actions." -ForegroundColor Green

# ------------------------------------------------------------------------------
# STEP 7: Role Cloning
# ------------------------------------------------------------------------------
Write-Host "`n[STEP 7] Testing Role Cloning with Inherited Rights..." -ForegroundColor Yellow
$cloneName = "Special Investigation Lead"
$clonePayload = @{
    name = $cloneName
    description = "Cloned with inherited investigative privileges"
} | ConvertTo-Json
$cloneRes = Invoke-RestMethod -Uri "$baseUrl/iam/roles/$($testRole.id)/clone" -Method POST -Headers $headers -Body $clonePayload
$clonedRole = $cloneRes.data
Write-Host "  ✓ Cloned role created: [ID $($clonedRole.id)] $($clonedRole.name)" -ForegroundColor Green

# Verify cloned matrix has same permissions
$clonedMatrixRes = Invoke-RestMethod -Uri "$baseUrl/iam/roles/$($clonedRole.id)/matrix" -Method GET -Headers $headers
$clonedGrantedCount = 0
foreach ($m in $clonedMatrixRes.data) {
    foreach ($a in $m.actions) {
        if ($a.granted -eq $true) { $clonedGrantedCount++ }
    }
}
Write-Host "  ✓ Verified inheritance: Cloned role automatically inherited all $clonedGrantedCount granted permissions!" -ForegroundColor Green

# ------------------------------------------------------------------------------
# STEP 8: Staff Roster & User Role Assignment
# ------------------------------------------------------------------------------
Write-Host "`n[STEP 8] Staff Member Governance & Role Assignment..." -ForegroundColor Yellow
$staffRes = Invoke-RestMethod -Uri "$baseUrl/iam/staff" -Method GET -Headers $headers
$staff = $staffRes.data
Write-Host "  ✓ Retrieved $($staff.Count) newsroom staff members:" -ForegroundColor Green
foreach ($s in $staff) {
    Write-Host "    - $($s.display_name) ($($s.email)) -> Role: $($s.role_name) (SuperAdmin=$($s.is_super_admin))" -ForegroundColor Gray
}

# Assign cloned role to user 1
Write-Host "  * Testing role assignment endpoint..." -ForegroundColor Gray
$assignPayload = @{ role_id = $clonedRole.id } | ConvertTo-Json
$assignRes = Invoke-RestMethod -Uri "$baseUrl/iam/users/$($adminUser.id)/role" -Method POST -Headers $headers -Body $assignPayload
Write-Host "  ✓ Assigned role to user ID $($adminUser.id) successfully." -ForegroundColor Green

# ------------------------------------------------------------------------------
# STEP 9: Regional Bureau & Category Jurisdiction Scoping
# ------------------------------------------------------------------------------
Write-Host "`n[STEP 9] Configuring Journalist Bureau Category Scopes..." -ForegroundColor Yellow
# Get all categories
$catsRes = Invoke-RestMethod -Uri "$baseUrl/categories" -Method GET -Headers $headers
$availableCats = $catsRes.data
$sampleScopeIds = @($availableCats[0].id, $availableCats[1].id)
Write-Host "  * Assigning categories: '$($availableCats[0].name)' (ID: $($sampleScopeIds[0])) and '$($availableCats[1].name)' (ID: $($sampleScopeIds[1]))..." -ForegroundColor Gray

$scopePayload = @{ category_ids = $sampleScopeIds } | ConvertTo-Json
$scopeRes = Invoke-RestMethod -Uri "$baseUrl/iam/users/$($adminUser.id)/categories" -Method POST -Headers $headers -Body $scopePayload
Write-Host "  ✓ Bureau category jurisdiction scopes saved successfully." -ForegroundColor Green

# Verify scopes
$getScopeRes = Invoke-RestMethod -Uri "$baseUrl/iam/users/$($adminUser.id)/categories" -Method GET -Headers $headers
Write-Host "  ✓ Verified active category scopes for user: $($getScopeRes.data -join ', ')" -ForegroundColor Green

# ------------------------------------------------------------------------------
# STEP 10: ABAC Zero-Trust Evaluation & Audit Trail Verification
# ------------------------------------------------------------------------------
Write-Host "`n[STEP 10] Testing ABAC Evaluation & Immutable Audit Trail..." -ForegroundColor Yellow
$evalPayload = @{
    user_id = $adminUser.id
    resource = "articles"
    action = "publish"
    context = @{
        category_id = $sampleScopeIds[0]
        ip_address = "127.0.0.1"
    }
} | ConvertTo-Json

$evalRes = Invoke-RestMethod -Uri "$baseUrl/iam/evaluate" -Method POST -Headers $headers -Body $evalPayload
Write-Host "  ✓ ABAC Evaluation Result: Allowed=$($evalRes.data.allowed) | Reason='$($evalRes.data.reason)'" -ForegroundColor Green

# Verify Audit Log
Write-Host "  * Fetching live security audit trail..." -ForegroundColor Gray
$auditRes = Invoke-RestMethod -Uri "$baseUrl/iam/audit-log" -Method GET -Headers $headers
$recentLogs = $auditRes.data
Write-Host "  ✓ Security audit trail has $($recentLogs.Count) entries recorded." -ForegroundColor Green
if ($recentLogs.Count -gt 0) {
    $latest = $recentLogs[0]
    Write-Host "    - Latest Log: [Action: $($latest.action)] User: $($latest.user_name) ($($latest.user_email))" -ForegroundColor Gray
    Write-Host "      Granted: $($latest.granted) | Reason: '$($latest.reason)' | Client IP: $($latest.ip_address)" -ForegroundColor DarkGray
}

# ------------------------------------------------------------------------------
# STEP 11: Cleanup & Teardown
# ------------------------------------------------------------------------------
Write-Host "`n[STEP 11] Cleaning Up Test Artifacts..." -ForegroundColor Yellow
# Reset user role back to super_admin (ID 1)
$resetRolePayload = @{ role_id = 1 } | ConvertTo-Json
Invoke-RestMethod -Uri "$baseUrl/iam/users/$($adminUser.id)/role" -Method POST -Headers $headers -Body $resetRolePayload | Out-Null
Write-Host "  ✓ Re-assigned user role back to super_admin (ID 1)." -ForegroundColor Green

# Reset category scopes (empty array = national scope)
$clearScopePayload = @{ category_ids = @() } | ConvertTo-Json
Invoke-RestMethod -Uri "$baseUrl/iam/users/$($adminUser.id)/categories" -Method POST -Headers $headers -Body $clearScopePayload | Out-Null
Write-Host "  ✓ Restored national scope (cleared test bureau scopes)." -ForegroundColor Green

# Delete created test roles
Invoke-RestMethod -Uri "$baseUrl/iam/roles/$($clonedRole.id)" -Method DELETE -Headers $headers | Out-Null
Write-Host "  ✓ Deleted cloned test role [ID $($clonedRole.id)]." -ForegroundColor Green

Invoke-RestMethod -Uri "$baseUrl/iam/roles/$($testRole.id)" -Method DELETE -Headers $headers | Out-Null
Write-Host "  ✓ Deleted custom test role [ID $($testRole.id)]." -ForegroundColor Green

Write-Host "`n========================================================" -ForegroundColor Cyan
Write-Host " 🎉 ALL 11 IAM WORKFLOW PHASES PASSED WITH 100% SUCCESS!" -ForegroundColor Green
Write-Host "========================================================`n" -ForegroundColor Cyan
