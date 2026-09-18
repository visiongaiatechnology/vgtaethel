// STATUS: DIAMANT VGT SUPREME
// Module: go-aethel/sphere/planner/planner_engine_test.go

package planner

import (
	"testing"

	"go-aethel/sphere"
)

func TestPlannerEngineProgressAndDependencies(t *testing.T) {
	dir := t.TempDir()
	store, err := sphere.NewObjectStore(dir)
	if err != nil {
		t.Fatalf("NewObjectStore failed: %v", err)
	}

	engine := NewEngine(store)

	plan := &sphere.PlanObject{
		Title:     "Aethel Beta 4 Rollout",
		Objective: "Sphere 2.0 veröffentlichen und Sicherheitsaudit bestehen",
		Milestones: []sphere.PlanMilestone{
			{ID: "M1", Title: "Sphere 2.0 Backend Architektur", Completed: true, Order: 1},
			{ID: "M2", Title: "VGT Writer & Browser 2.0", Completed: false, Order: 2},
		},
		Tasks: []sphere.PlanTask{
			{ID: "T1", MilestoneID: "M1", Title: "Object Store & Engines bauen", Status: "completed"},
			{ID: "T2", MilestoneID: "M2", Title: "Frontend Window Manager refactoren", Status: "in_progress", DependsOnIDs: []string{"T1"}},
		},
	}

	created, err := engine.CreatePlan(plan)
	if err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}
	// 2 completed out of 4 items -> 50%
	if created.ProgressPct != 50 {
		t.Fatalf("expected 50%% progress, got %d%%", created.ProgressPct)
	}

	// Adding invalid dependency should fail
	badTask := sphere.PlanTask{
		ID:           "T3",
		Title:        "Unmögliche Aufgabe",
		DependsOnIDs: []string{"NON_EXISTENT_TASK"},
	}
	if err := engine.AddTask(created.ID, badTask); err == nil {
		t.Fatal("expected error on non-existent dependency")
	}
}
