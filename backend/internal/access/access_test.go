package access

import "testing"

func TestPolicyUsesMostSpecificRule(t *testing.T) {
	policy := Policy{
		Default: Parse("R"),
		Rules: []Rule{
			{Pattern: "/private/**", Permissions: Parse("")},
			{Pattern: "/private/public/**", Permissions: Parse("RW")},
		},
	}
	if got := policy.For("/private/secret.txt"); got.Read {
		t.Fatal("private path unexpectedly readable")
	}
	if got := policy.For("/private/public/readme.txt"); !got.Read || !got.Write || got.Delete {
		t.Fatalf("permissions = %#v", got)
	}
}

func TestWildcards(t *testing.T) {
	policy := Policy{Rules: []Rule{{Pattern: "/teams/*/public/**", Permissions: Parse("R")}}}
	if !policy.For("/teams/red/public/docs/a.txt").Read {
		t.Fatal("expected wildcard match")
	}
	if policy.For("/teams/red/private/a.txt").Read {
		t.Fatal("unexpected wildcard match")
	}
}
