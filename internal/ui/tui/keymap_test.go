package tui

import (
	"reflect"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
)

func bindings(k KeyMap) map[string]key.Binding {
	out := make(map[string]key.Binding)

	v := reflect.ValueOf(k)
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		if binding, ok := v.Field(i).Interface().(key.Binding); ok {
			out[field.Name] = binding
		}
	}

	return out
}

func TestNoKeyIsBoundTwice(t *testing.T) {
	exempt := map[string]bool{"ForceQuit": true}

	owner := make(map[string]string)

	for name, binding := range bindings(DefaultKeys) {
		if exempt[name] {
			continue
		}
		for _, k := range binding.Keys() {
			if previous, taken := owner[k]; taken {
				t.Errorf("key %q is bound to both %s and %s", k, previous, name)
				continue
			}
			owner[k] = name
		}
	}
}

func TestEscapeGoesBackAndDoesNotQuit(t *testing.T) {
	if !containsKey(DefaultKeys.Back, "esc") {
		t.Error("Back is not bound to esc")
	}
	if containsKey(DefaultKeys.Quit, "esc") {
		t.Error("Quit is bound to esc, which makes Escape close the app instead of going back")
	}
}

func TestEveryBindingHasHelp(t *testing.T) {
	for name, binding := range bindings(DefaultKeys) {
		help := binding.Help()
		if help.Key == "" {
			t.Errorf("%s has no help key", name)
		}
		if help.Desc == "" {
			t.Errorf("%s has no help description", name)
		}
	}
}

func TestHelpBarKeysComeFromTheBindings(t *testing.T) {
	shortcuts := DefaultKeys.listShortcuts()
	if len(shortcuts) == 0 {
		t.Fatal("the help bar lists no shortcuts")
	}

	for _, s := range shortcuts {
		if s.Keys == "" {
			t.Errorf("shortcut %q has no keys", s.Label)
		}
		if s.Label == "" {
			t.Errorf("shortcut with keys %q has no label", s.Keys)
		}
	}

	byLabel := make(map[string]string, len(shortcuts))
	for _, s := range shortcuts {
		byLabel[s.Label] = s.Keys
	}

	wantLogs := DefaultKeys.Logs.Help().Key
	found := false
	for _, keys := range byLabel {
		if keys == wantLogs {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no help entry uses the Logs key %q", wantLogs)
	}
}

func TestLetterShortcutsExistForTheList(t *testing.T) {
	for _, binding := range []key.Binding{
		DefaultKeys.Logs, DefaultKeys.Copy, DefaultKeys.Add,
		DefaultKeys.Edit, DefaultKeys.Delete, DefaultKeys.Settings,
	} {
		keys := binding.Keys()
		if len(keys) == 0 {
			t.Fatalf("%v has no keys", binding.Help())
		}
		if len(keys[0]) != 1 || !strings.ContainsAny(keys[0], "abcdefghijklmnopqrstuvwxyz") {
			t.Errorf("expected a plain letter shortcut, got %q", keys[0])
		}
	}
}

func containsKey(binding key.Binding, want string) bool {
	for _, k := range binding.Keys() {
		if k == want {
			return true
		}
	}
	return false
}
