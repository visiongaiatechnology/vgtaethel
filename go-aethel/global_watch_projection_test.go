package main

import (
	_ "embed"
	"math"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

//go:embed frontend/modules/global_watch_projection.js
var globalWatchProjectionJS []byte

func TestGlobalWatchProjectionHasNoNorthSouthDrift(t *testing.T) {
	vm := goja.New()
	source := strings.ReplaceAll(string(globalWatchProjectionJS), "export ", "")
	if _, err := vm.RunString(source); err != nil {
		t.Fatalf("load projection module: %v", err)
	}
	createProjection, ok := goja.AssertFunction(vm.Get("createProjection"))
	if !ok {
		t.Fatal("createProjection is not callable")
	}
	project, ok := goja.AssertFunction(vm.Get("project"))
	if !ok {
		t.Fatal("project is not callable")
	}
	unproject, ok := goja.AssertFunction(vm.Get("unproject"))
	if !ok {
		t.Fatal("unproject is not callable")
	}

	cameras := [][3]float64{
		{0, 0, 1},
		{-10 * math.Pi / 180, 50 * math.Pi / 180, 1.55},
		{70 * math.Pi / 180, -55 * math.Pi / 180, 1.2},
	}
	for _, camera := range cameras {
		projectionValue, err := createProjection(goja.Undefined(), vm.ToValue(camera[0]), vm.ToValue(camera[1]), vm.ToValue(camera[2]), vm.ToValue(1200), vm.ToValue(800))
		if err != nil {
			t.Fatalf("create projection: %v", err)
		}
		for latitude := -80.0; latitude <= 80; latitude += 5 {
			for longitude := -180.0; longitude < 180; longitude += 5 {
				screenValue, err := project(goja.Undefined(), projectionValue, vm.ToValue(latitude), vm.ToValue(longitude))
				if err != nil {
					t.Fatalf("project %.1f/%.1f: %v", latitude, longitude, err)
				}
				screen := screenValue.ToObject(vm)
				if !screen.Get("visible").ToBoolean() || screen.Get("depth").ToFloat() < 1e-7 {
					continue
				}
				geoValue, err := unproject(
					goja.Undefined(),
					projectionValue,
					vm.ToValue(screen.Get("x").ToFloat()-0.5),
					vm.ToValue(screen.Get("y").ToFloat()-0.5),
				)
				if err != nil || goja.IsNull(geoValue) || goja.IsUndefined(geoValue) {
					t.Fatalf("unproject %.1f/%.1f: %v", latitude, longitude, err)
				}
				geo := geoValue.ToObject(vm)
				latitudeError := math.Abs(geo.Get("lat").ToFloat() - latitude)
				longitudeError := math.Abs(geo.Get("lon").ToFloat() - longitude)
				longitudeError = math.Min(longitudeError, 360-longitudeError)
				if latitudeError > 1e-6 || longitudeError > 1e-6 {
					t.Fatalf("projection drift at %.1f/%.1f: lat=%g lon=%g", latitude, longitude, latitudeError, longitudeError)
				}
			}
		}
	}
}

func TestGlobalWatchRejectsNonEquirectangularTexture(t *testing.T) {
	vm := goja.New()
	source := strings.ReplaceAll(string(globalWatchProjectionJS), "export ", "")
	if _, err := vm.RunString(source); err != nil {
		t.Fatalf("load projection module: %v", err)
	}
	validate, ok := goja.AssertFunction(vm.Get("validateEquirectangularDimensions"))
	if !ok {
		t.Fatal("validateEquirectangularDimensions is not callable")
	}
	for _, test := range []struct {
		width, height int
		valid         bool
	}{{8192, 4096, true}, {2048, 1024, true}, {1920, 704, false}, {10, 5, false}} {
		value, err := validate(goja.Undefined(), vm.ToValue(test.width), vm.ToValue(test.height))
		if err != nil {
			t.Fatalf("validate %dx%d: %v", test.width, test.height, err)
		}
		if got := value.ToObject(vm).Get("valid").ToBoolean(); got != test.valid {
			t.Fatalf("validate %dx%d = %v, want %v", test.width, test.height, got, test.valid)
		}
	}
}
