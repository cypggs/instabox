package store

import (
	"path/filepath"
	"testing"

	"github.com/ibox/ibox/internal/model"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// ensureOrg creates a user + personal org so machines can reference it.
func ensureOrg(t *testing.T, s *Store, email string) string {
	t.Helper()
	uid, err := s.UpsertUser(email, "T", "x", false)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	orgID, err := s.EnsurePersonalOrg(uid, email)
	if err != nil {
		t.Fatalf("org: %v", err)
	}
	return orgID
}

func TestMachineCRUDAndNameUnique(t *testing.T) {
	s := testStore(t)
	orgID := ensureOrg(t, s, "a@b.c")
	tmpl := &model.TemplateRef{ID: "minimal", Version: "2026.11.1"}
	m := &model.Machine{ID: "m_test1", OrgID: orgID, Name: "box1", DesiredState: model.DesiredRunning, State: model.StateCreating, Template: tmpl}
	if err := s.InsertMachine(m); err != nil {
		t.Fatalf("insert: %v", err)
	}
	got, err := s.GetMachineByID("m_test1")
	if err != nil || got.Name != "box1" {
		t.Fatalf("get: %v %+v", err, got)
	}
	// duplicate name in same org must fail
	dup := &model.Machine{ID: "m_test2", OrgID: "org_1", Name: "box1", Template: tmpl}
	if err := s.InsertMachine(dup); err == nil {
		t.Fatal("expected UNIQUE violation for duplicate name")
	}
	// same name in another org is fine
	org2 := ensureOrg(t, s, "d@e.f")
	other := &model.Machine{ID: "m_test3", OrgID: org2, Name: "box1", Template: tmpl}
	if err := s.InsertMachine(other); err != nil {
		t.Fatalf("cross-org same name: %v", err)
	}
	// update with revision bump
	if err := s.UpdateMachine("m_test1", func(mm *model.Machine) { mm.State = model.StateRunning }); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ = s.GetMachineByID("m_test1")
	if got.State != model.StateRunning {
		t.Fatalf("update did not apply: %+v", got)
	}
	// delete removes the row and dependents
	if err := s.DeleteMachine("m_test1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.GetMachineByID("m_test1"); err == nil {
		t.Fatal("expected not found after delete")
	}
}

func TestIdempotencyKey(t *testing.T) {
	s := testStore(t)
	orgID := ensureOrg(t, s, "a@b.c")
	tmpl := &model.TemplateRef{ID: "minimal", Version: "2026.11.1"}
	if err := s.InsertMachine(&model.Machine{ID: "m_x", OrgID: orgID, Name: "boxx", Template: tmpl}); err != nil {
		t.Fatalf("insert machine: %v", err)
	}
	op := &model.Operation{ID: "op_key-1", MachineID: "m_x", Kind: "create", Status: "pending"}
	if err := s.InsertOperation(op); err != nil {
		t.Fatalf("insert op: %v", err)
	}
	got, err := s.FindOperationByIdempotencyKey("op_key-1")
	if err != nil || got == nil || got.ID != "op_key-1" {
		t.Fatalf("lookup: %v %+v", err, got)
	}
	// replaying the same idempotency key must be rejected (caller returns the
	// recorded operation instead of creating a new one)
	dup := &model.Operation{ID: "op_key-1", MachineID: "m_x", Kind: "create", Status: "pending"}
	if err := s.InsertOperation(dup); err == nil {
		t.Fatal("expected UNIQUE violation for duplicate idempotency key")
	}
}

func TestMachineLock(t *testing.T) {
	s := testStore(t)
	orgID := ensureOrg(t, s, "a@b.c")
	tmpl := &model.TemplateRef{ID: "minimal", Version: "2026.11.1"}
	m := &model.Machine{ID: "m_l", OrgID: orgID, Name: "locked", Template: tmpl}
	if err := s.InsertMachine(m); err != nil {
		t.Fatalf("insert: %v", err)
	}
	ok, err := s.AcquireMachineLock("m_l", "op_a")
	if err != nil || !ok {
		t.Fatalf("first acquire: %v %v", ok, err)
	}
	ok, _ = s.AcquireMachineLock("m_l", "op_b")
	if ok {
		t.Fatal("second acquire must fail while held")
	}
	if err := s.ReleaseMachineLock("m_l"); err != nil {
		t.Fatalf("release: %v", err)
	}
	ok, _ = s.AcquireMachineLock("m_l", "op_b")
	if !ok {
		t.Fatal("acquire after release must succeed")
	}
}
