package domain

import (
	"testing"
)

func TestThreshold(t *testing.T) {
	t.Run("NewThreshold creates valid threshold", func(t *testing.T) {
		cases := []struct {
			value float64
			valid bool
		}{
			{0, true},
			{50, true},
			{100, true},
			{-1, false},
			{101, false},
		}

		for _, tc := range cases {
			_, err := NewThreshold(tc.value)
			if tc.valid && err != nil {
				t.Errorf("NewThreshold(%v) should be valid, got error: %v", tc.value, err)
			}
			if !tc.valid && err == nil {
				t.Errorf("NewThreshold(%v) should be invalid, got no error", tc.value)
			}
		}
	})

	t.Run("MustThreshold panics on invalid value", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustThreshold(-1) should panic")
			}
		}()
		MustThreshold(-1)
	})

	t.Run("IsMet returns true when coverage meets threshold", func(t *testing.T) {
		threshold := MustThreshold(80)
		if !threshold.IsMet(80) {
			t.Error("80% should meet 80% threshold")
		}
		if !threshold.IsMet(85) {
			t.Error("85% should meet 80% threshold")
		}
		if threshold.IsMet(79) {
			t.Error("79% should not meet 80% threshold")
		}
	})

	t.Run("Shortfall calculates correctly", func(t *testing.T) {
		threshold := MustThreshold(80)
		if shortfall := threshold.Shortfall(75); shortfall != 5.0 {
			t.Errorf("Expected shortfall of 5.0, got %v", shortfall)
		}
		if shortfall := threshold.Shortfall(80); shortfall != 0 {
			t.Errorf("Expected shortfall of 0, got %v", shortfall)
		}
		if shortfall := threshold.Shortfall(85); shortfall != 0 {
			t.Errorf("Expected shortfall of 0, got %v", shortfall)
		}
	})

	t.Run("ThresholdFromPtr uses default when nil", func(t *testing.T) {
		threshold := ThresholdFromPtr(nil, 80)
		if threshold.Value() != 80 {
			t.Errorf("Expected 80, got %v", threshold.Value())
		}

		val := 90.0
		threshold = ThresholdFromPtr(&val, 80)
		if threshold.Value() != 90 {
			t.Errorf("Expected 90, got %v", threshold.Value())
		}
	})

	t.Run("Equals compares thresholds", func(t *testing.T) {
		t1 := MustThreshold(80)
		t2 := MustThreshold(80)
		t3 := MustThreshold(90)

		if !t1.Equals(t2) {
			t.Error("80% should equal 80%")
		}
		if t1.Equals(t3) {
			t.Error("80% should not equal 90%")
		}
	})

	t.Run("String formats correctly", func(t *testing.T) {
		threshold := MustThreshold(80.5)
		if s := threshold.String(); s != "80.5%" {
			t.Errorf("Expected '80.5%%', got '%s'", s)
		}
	})

	t.Run("IsExceededBy returns true when coverage exceeds threshold", func(t *testing.T) {
		threshold := MustThreshold(80)
		if !threshold.IsExceededBy(81) {
			t.Error("81% should exceed 80% threshold")
		}
		if threshold.IsExceededBy(80) {
			t.Error("80% should not exceed 80% threshold")
		}
		if threshold.IsExceededBy(79) {
			t.Error("79% should not exceed 80% threshold")
		}
	})

	t.Run("ZeroThreshold returns 0%", func(t *testing.T) {
		threshold := ZeroThreshold()
		if threshold.Value() != 0 {
			t.Errorf("Expected 0, got %v", threshold.Value())
		}
	})

	t.Run("Ptr returns pointer to value", func(t *testing.T) {
		threshold := MustThreshold(80)
		ptr := threshold.Ptr()
		if ptr == nil {
			t.Fatal("Expected non-nil pointer")
		}
		if *ptr != 80 {
			t.Errorf("Expected 80, got %v", *ptr)
		}
	})
}

func TestDomainName(t *testing.T) {
	t.Run("NewDomainName creates valid name", func(t *testing.T) {
		name, err := NewDomainName("core")
		if err != nil {
			t.Errorf("NewDomainName('core') failed: %v", err)
		}
		if name.String() != "core" {
			t.Errorf("Expected 'core', got '%s'", name.String())
		}
	})

	t.Run("NewDomainName trims whitespace", func(t *testing.T) {
		name, err := NewDomainName("  core  ")
		if err != nil {
			t.Errorf("NewDomainName failed: %v", err)
		}
		if name.String() != "core" {
			t.Errorf("Expected 'core', got '%s'", name.String())
		}
	})

	t.Run("NewDomainName rejects empty name", func(t *testing.T) {
		_, err := NewDomainName("")
		if err != ErrEmptyDomainName {
			t.Errorf("Expected ErrEmptyDomainName, got %v", err)
		}

		_, err = NewDomainName("   ")
		if err != ErrEmptyDomainName {
			t.Errorf("Expected ErrEmptyDomainName for whitespace, got %v", err)
		}
	})

	t.Run("MustDomainName panics on empty", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustDomainName('') should panic")
			}
		}()
		MustDomainName("")
	})

	t.Run("Equals compares names", func(t *testing.T) {
		n1 := MustDomainName("core")
		n2 := MustDomainName("core")
		n3 := MustDomainName("api")

		if !n1.Equals(n2) {
			t.Error("'core' should equal 'core'")
		}
		if n1.Equals(n3) {
			t.Error("'core' should not equal 'api'")
		}
	})

	t.Run("IsEmpty returns true for zero value", func(t *testing.T) {
		var name DomainName
		if !name.IsEmpty() {
			t.Error("Zero value should be empty")
		}

		name = MustDomainName("core")
		if name.IsEmpty() {
			t.Error("'core' should not be empty")
		}
	})
}

func TestFilePath(t *testing.T) {
	t.Run("NewFilePath creates valid path", func(t *testing.T) {
		path, err := NewFilePath("/path/to/file.go")
		if err != nil {
			t.Errorf("NewFilePath failed: %v", err)
		}
		if path.String() != "/path/to/file.go" {
			t.Errorf("Unexpected path: %s", path.String())
		}
	})

	t.Run("NewFilePath normalizes path", func(t *testing.T) {
		path, err := NewFilePath("/path//to/../to/file.go")
		if err != nil {
			t.Errorf("NewFilePath failed: %v", err)
		}
		if path.String() != "/path/to/file.go" {
			t.Errorf("Expected normalized path, got: %s", path.String())
		}
	})

	t.Run("NewFilePath rejects empty path", func(t *testing.T) {
		_, err := NewFilePath("")
		if err != ErrEmptyFilePath {
			t.Errorf("Expected ErrEmptyFilePath, got %v", err)
		}
	})

	t.Run("Dir returns directory portion", func(t *testing.T) {
		path := MustFilePath("/path/to/file.go")
		dir := path.Dir()
		if dir.String() != "/path/to" {
			t.Errorf("Expected '/path/to', got '%s'", dir.String())
		}
	})

	t.Run("Base returns filename", func(t *testing.T) {
		path := MustFilePath("/path/to/file.go")
		base := path.Base()
		if base != "file.go" {
			t.Errorf("Expected 'file.go', got '%s'", base)
		}
	})

	t.Run("HasPrefix checks path prefix", func(t *testing.T) {
		path := MustFilePath("/path/to/file.go")
		if !path.HasPrefix("/path") {
			t.Error("Should have prefix '/path'")
		}
		if path.HasPrefix("/other") {
			t.Error("Should not have prefix '/other'")
		}
	})

	t.Run("Equals compares paths", func(t *testing.T) {
		p1 := MustFilePath("/path/to/file.go")
		p2 := MustFilePath("/path/to/file.go")
		p3 := MustFilePath("/other/file.go")

		if !p1.Equals(p2) {
			t.Error("Same paths should be equal")
		}
		if p1.Equals(p3) {
			t.Error("Different paths should not be equal")
		}
	})

	t.Run("IsEmpty returns true for zero value", func(t *testing.T) {
		var path FilePath
		if !path.IsEmpty() {
			t.Error("Zero value should be empty")
		}

		path = MustFilePath("/path/to/file.go")
		if path.IsEmpty() {
			t.Error("Valid path should not be empty")
		}
	})

	t.Run("MatchesPattern checks glob pattern", func(t *testing.T) {
		path := MustFilePath("service_test.go")
		if !path.MatchesPattern("*_test.go") {
			t.Error("Should match *_test.go pattern")
		}
		if path.MatchesPattern("*.txt") {
			t.Error("Should not match *.txt pattern")
		}
	})

	t.Run("MatchesAnyPattern checks multiple patterns", func(t *testing.T) {
		path := MustFilePath("service_test.go")
		if !path.MatchesAnyPattern([]string{"*.txt", "*_test.go"}) {
			t.Error("Should match one of the patterns")
		}
		if path.MatchesAnyPattern([]string{"*.txt", "*.md"}) {
			t.Error("Should not match any pattern")
		}
		if path.MatchesAnyPattern([]string{}) {
			t.Error("Should not match empty pattern list")
		}
	})

	t.Run("MustFilePath panics on empty", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustFilePath('') should panic")
			}
		}()
		MustFilePath("")
	})
}

func TestPercentage(t *testing.T) {
	t.Run("NewPercentage rounds to one decimal", func(t *testing.T) {
		p := NewPercentage(80.123)
		if p.Value() != 80.1 {
			t.Errorf("Expected 80.1, got %v", p.Value())
		}

		p = NewPercentage(80.15)
		if p.Value() != 80.2 {
			t.Errorf("Expected 80.2, got %v", p.Value())
		}
	})

	t.Run("PercentageFromRatio calculates correctly", func(t *testing.T) {
		p := PercentageFromRatio(80, 100)
		if p.Value() != 80.0 {
			t.Errorf("Expected 80.0, got %v", p.Value())
		}

		p = PercentageFromRatio(1, 3)
		if p.Value() != 33.3 {
			t.Errorf("Expected 33.3, got %v", p.Value())
		}

		p = PercentageFromRatio(0, 0)
		if p.Value() != 0 {
			t.Errorf("Expected 0 for 0/0, got %v", p.Value())
		}
	})

	t.Run("Comparison methods work correctly", func(t *testing.T) {
		p1 := NewPercentage(80)
		p2 := NewPercentage(90)
		p3 := NewPercentage(80)

		if !p2.GreaterThan(p1) {
			t.Error("90 should be greater than 80")
		}
		if !p1.LessThan(p2) {
			t.Error("80 should be less than 90")
		}
		if !p1.Equals(p3) {
			t.Error("80 should equal 80")
		}
	})

	t.Run("MeetsThreshold checks against threshold", func(t *testing.T) {
		p := NewPercentage(85)
		threshold := MustThreshold(80)

		if !p.MeetsThreshold(threshold) {
			t.Error("85% should meet 80% threshold")
		}

		lowP := NewPercentage(75)
		if lowP.MeetsThreshold(threshold) {
			t.Error("75% should not meet 80% threshold")
		}
	})

	t.Run("Delta calculates difference", func(t *testing.T) {
		p1 := NewPercentage(80)
		p2 := NewPercentage(70)

		delta := p1.Delta(p2)
		if delta != 10.0 {
			t.Errorf("Expected delta of 10.0, got %v", delta)
		}
	})

	t.Run("String formats correctly", func(t *testing.T) {
		p := NewPercentage(80.5)
		if s := p.String(); s != "80.5%" {
			t.Errorf("Expected '80.5%%', got '%s'", s)
		}
	})

	t.Run("IsZero returns true for zero value", func(t *testing.T) {
		p := NewPercentage(0)
		if !p.IsZero() {
			t.Error("0% should be zero")
		}

		p = NewPercentage(80)
		if p.IsZero() {
			t.Error("80% should not be zero")
		}
	})

	t.Run("Difference calculates absolute difference", func(t *testing.T) {
		p1 := NewPercentage(80)
		p2 := NewPercentage(70)

		diff := p1.Difference(p2)
		if diff != 10.0 {
			t.Errorf("Expected difference of 10.0, got %v", diff)
		}

		// Difference is absolute value (same as Delta)
		diff = p2.Difference(p1)
		if diff != -10.0 {
			t.Errorf("Expected difference of -10.0, got %v", diff)
		}
	})
}
