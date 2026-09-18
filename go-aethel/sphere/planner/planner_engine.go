// STATUS: DIAMANT VGT SUPREME
// Module: go-aethel/sphere/planner/planner_engine.go
// Purpose: Master Planner Engine, Milestones, Task Dependencies, and Agent Run Integration

package planner

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"go-aethel/sphere"
)

type Engine struct {
	store *sphere.ObjectStore
}

func NewEngine(store *sphere.ObjectStore) *Engine {
	return &Engine{store: store}
}

// CreatePlan initializes a new PlanObject.
func (e *Engine) CreatePlan(plan *sphere.PlanObject) (*sphere.PlanObject, error) {
	if plan == nil {
		return nil, errors.New("plan cannot be nil")
	}
	plan.Title = strings.TrimSpace(plan.Title)
	if plan.Title == "" {
		return nil, errors.New("plan title is required")
	}
	if plan.ID == "" {
		plan.ID = sphere.GenerateID("PLAN")
	}
	if plan.Status == "" {
		plan.Status = "draft"
	}
	now := time.Now().UTC()
	plan.CreatedAt = now
	plan.UpdatedAt = now
	plan.ProgressPct = e.CalculateProgress(plan)

	data, err := json.Marshal(plan)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal plan data: %w", err)
	}

	obj := &sphere.SphereObject{
		ID:        plan.ID,
		Type:      sphere.TypePlan,
		Title:     plan.Title,
		Summary:   fmt.Sprintf("Ziel: %s · %d%% Fortschritt", plan.Objective, plan.ProgressPct),
		Desktop:   sphere.DesktopWork,
		Tags:      []string{"plan", "goals", "milestones"},
		Data:      data,
		CreatedAt: now,
		UpdatedAt: now,
		SourceApp: "planner",
	}

	if err := e.store.Put(obj); err != nil {
		return nil, err
	}
	return plan, nil
}

// GetPlan loads a PlanObject by ID.
func (e *Engine) GetPlan(id string) (*sphere.PlanObject, error) {
	obj, exists := e.store.Get(id)
	if !exists || obj.Type != sphere.TypePlan {
		return nil, fmt.Errorf("plan %s not found", id)
	}
	var plan sphere.PlanObject
	if err := json.Unmarshal(obj.Data, &plan); err != nil {
		return nil, fmt.Errorf("failed to deserialize plan data: %w", err)
	}
	return &plan, nil
}

// ListPlans loads all plans.
func (e *Engine) ListPlans() ([]*sphere.PlanObject, error) {
	objects := e.store.List(sphere.TypePlan, "", "")
	plans := make([]*sphere.PlanObject, 0, len(objects))
	for _, obj := range objects {
		var plan sphere.PlanObject
		if err := json.Unmarshal(obj.Data, &plan); err == nil {
			plans = append(plans, &plan)
		}
	}
	return plans, nil
}

// UpdatePlan saves changes to a plan.
func (e *Engine) UpdatePlan(plan *sphere.PlanObject) error {
	if plan == nil || plan.ID == "" {
		return errors.New("invalid plan object")
	}
	plan.UpdatedAt = time.Now().UTC()
	plan.ProgressPct = e.CalculateProgress(plan)

	data, err := json.Marshal(plan)
	if err != nil {
		return fmt.Errorf("failed to marshal plan data: %w", err)
	}

	obj, exists := e.store.Get(plan.ID)
	if !exists {
		obj = &sphere.SphereObject{
			ID:        plan.ID,
			Type:      sphere.TypePlan,
			CreatedAt: plan.CreatedAt,
			Desktop:   sphere.DesktopWork,
			SourceApp: "planner",
		}
	}
	obj.Title = plan.Title
	obj.Summary = fmt.Sprintf("Ziel: %s · %d%% Fortschritt", plan.Objective, plan.ProgressPct)
	obj.Data = data
	obj.UpdatedAt = plan.UpdatedAt

	return e.store.Put(obj)
}

// CalculateProgress calculates the completion percentage across all tasks and milestones.
func (e *Engine) CalculateProgress(plan *sphere.PlanObject) int {
	totalItems := len(plan.Milestones) + len(plan.Tasks)
	if totalItems == 0 {
		return 0
	}
	completed := 0
	for _, m := range plan.Milestones {
		if m.Completed {
			completed++
		}
	}
	for _, t := range plan.Tasks {
		if t.Status == "completed" {
			completed++
		}
	}
	return int((float64(completed) / float64(totalItems)) * 100.0)
}

// AddTask adds a task to a plan with dependency checks.
func (e *Engine) AddTask(planID string, task sphere.PlanTask) error {
	plan, err := e.GetPlan(planID)
	if err != nil {
		return err
	}
	if task.ID == "" {
		task.ID = fmt.Sprintf("TASK-%d", time.Now().UnixNano()%1000000)
	}
	if task.Status == "" {
		task.Status = "pending"
	}

	// Validate dependencies exist in the plan
	for _, depID := range task.DependsOnIDs {
		exists := false
		for _, existing := range plan.Tasks {
			if existing.ID == depID {
				exists = true
				break
			}
		}
		if !exists {
			return fmt.Errorf("dependent task %s does not exist in plan %s", depID, planID)
		}
	}

	plan.Tasks = append(plan.Tasks, task)
	return e.UpdatePlan(plan)
}
