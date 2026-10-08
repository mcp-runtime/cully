package cully

import "testing"

func TestSuggestTaskName(t *testing.T) {
	cases := map[string]string{
		"feat/oauth-validation":               "Oauth validation",
		"product/agent-workspace-positioning": "Agent workspace positioning",
		"fix_login-bug":                       "Fix login bug",
		"docs":                                "Docs",
		"feat/über-cool":                      "Über cool",
		"main":                                "",
		"master":                              "",
		"develop":                             "",
		"HEAD":                                "",
		"":                                    "",
		"feat/":                               "",
	}
	for branch, want := range cases {
		if got := suggestTaskName(branch); got != want {
			t.Errorf("suggestTaskName(%q) = %q, want %q", branch, got, want)
		}
	}
}
