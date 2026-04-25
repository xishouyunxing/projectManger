# Production Line Optional Process Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow production lines to be created and updated without requiring a process, while keeping the current front-end production line UI unchanged.

**Architecture:** Make `process_id` nullable in the `ProductionLine` model and treat the process association as optional in the backend. Keep the current front-end payload behavior intact by allowing omitted `process_id`, and verify with focused controller tests.

**Tech Stack:** Go, Gin, Gorm, Go testing

---

## File Structure

- Modify: `backend/models/production_line.go`
- Modify: `backend/controllers/production_line.go`
- Modify: `backend/controllers/production_line_test.go`

### Task 1: Add failing test for creating a production line without process_id

**Files:**
- Test: `backend/controllers/production_line_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestCreateProductionLineWithoutProcessID(t *testing.T) {
	setupProductionLineTestDB(t)
	admin, err := testutil.CreateAdminUser(database.DB, "AdminPass123")
	require.NoError(t, err)
	token, err := testutil.GenerateTestToken(admin.ID, "admin", config.AppConfig.JWTSecret)
	require.NoError(t, err)

	body := []byte(`{"name":"吊臂","code":"db","type":"upper","status":"active"}`)
	r := router.SetupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/production-lines", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), `"code":"db"`)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./controllers -run TestCreateProductionLineWithoutProcessID -count=1`
Expected: FAIL because `process_id` currently defaults to `0` and violates the foreign key.

### Task 2: Make process association optional

**Files:**
- Modify: `backend/models/production_line.go`
- Modify: `backend/controllers/production_line.go`

- [ ] **Step 1: Change the model to use nullable process fields**

```go
type ProductionLine struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:200;not null" json:"name"`
	Code        string         `gorm:"uniqueIndex;size:50;not null" json:"code"`
	Type        string         `gorm:"size:50;not null" json:"type"`
	Description string         `gorm:"type:text" json:"description"`
	Status      string         `gorm:"size:20;default:active" json:"status"`
	ProcessID   *uint          `gorm:"index" json:"process_id"`

	Process              *Process                    `json:"process,omitempty"`
	Programs             []Program                   `json:"programs,omitempty"`
	CustomFieldTemplates []ProductionLineCustomField `json:"custom_field_templates,omitempty"`
}
```

- [ ] **Step 2: Ensure create/update paths allow omitted process_id**

```go
var productionLine models.ProductionLine
if err := c.ShouldBindJSON(&productionLine); err != nil {
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	return
}

if err := database.DB.Create(&productionLine).Error; err != nil {
	c.JSON(http.StatusInternalServerError, gin.H{"error": "创建生产线失败"})
	return
}
```

The controller should not coerce omitted `process_id` into `0`, and updates should allow `process_id` to stay `NULL`.

- [ ] **Step 3: Run the focused test to verify it passes**

Run: `go test ./controllers -run TestCreateProductionLineWithoutProcessID -count=1`
Expected: PASS

### Task 3: Run targeted and full backend verification

**Files:**
- Modify: any files above if verification reveals issues

- [ ] **Step 1: Run production line controller tests**

Run: `go test ./controllers -run TestCreateProductionLine -count=1`
Expected: PASS

- [ ] **Step 2: Run full backend test suite**

Run: `go test ./...`
Expected: PASS

## Self-Review

### Spec coverage

Covered spec items:

1. `process_id` nullable in model: Task 2
2. create/update allow omitted process: Task 2
3. no default process data reintroduced: Task 2

### Placeholder scan

1. No TODO/TBD placeholders remain
2. Focused commands and files are explicit

### Type consistency

Consistent names used throughout:

1. `ProcessID *uint`
2. `Process *Process`
