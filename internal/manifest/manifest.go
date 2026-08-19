package manifest

import (
	"fmt"
	"os"
	"time"

	"example.com/qemu-arm-can-test/internal/socketcan"
	"gopkg.in/yaml.v3"
)

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	parsed, err := time.ParseDuration(node.Value)
	if err != nil {
		return fmt.Errorf("parse duration %q: %w", node.Value, err)
	}
	d.Duration = parsed
	return nil
}

func (d Duration) MarshalYAML() (any, error) {
	return d.String(), nil
}

type TestRun struct {
	Name      string     `yaml:"name" json:"name"`
	Interface string     `yaml:"interface" json:"interface"`
	Timeout   Duration   `yaml:"timeout" json:"timeout"`
	Evidence  []string   `yaml:"evidence" json:"evidence"`
	Cases     []TestCase `yaml:"cases" json:"cases"`
}

type TestCase struct {
	Name          string   `yaml:"name" json:"name"`
	Request       string   `yaml:"request" json:"request"`
	Expected      string   `yaml:"expected,omitempty" json:"expected,omitempty"`
	ExpectSilence bool     `yaml:"expectSilence,omitempty" json:"expect_silence,omitempty"`
	Timeout       Duration `yaml:"timeout,omitempty" json:"timeout"`
}

type PreparedCase struct {
	Name          string
	Request       socketcan.Frame
	Expected      socketcan.Frame
	ExpectSilence bool
	Timeout       time.Duration
}

func Load(path string) (TestRun, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return TestRun{}, nil, fmt.Errorf("read manifest %q: %w", path, err)
	}

	var run TestRun
	if err := yaml.Unmarshal(raw, &run); err != nil {
		return TestRun{}, nil, fmt.Errorf("decode manifest %q: %w", path, err)
	}
	return run, raw, nil
}

func (r TestRun) Validate() ([]PreparedCase, error) {
	if r.Name == "" {
		return nil, fmt.Errorf("manifest name is required")
	}
	if r.Interface == "" {
		return nil, fmt.Errorf("CAN interface is required")
	}
	if r.Timeout.Duration <= 0 {
		return nil, fmt.Errorf("run timeout must be positive")
	}
	if len(r.Cases) == 0 {
		return nil, fmt.Errorf("at least one test case is required")
	}

	prepared := make([]PreparedCase, 0, len(r.Cases))
	for index, testCase := range r.Cases {
		if testCase.Name == "" {
			return nil, fmt.Errorf("case %d: name is required", index+1)
		}
		request, err := socketcan.ParseFrame(testCase.Request)
		if err != nil {
			return nil, fmt.Errorf("case %q request: %w", testCase.Name, err)
		}
		if testCase.ExpectSilence == (testCase.Expected != "") {
			return nil, fmt.Errorf("case %q must set exactly one of expected or expectSilence", testCase.Name)
		}

		var expected socketcan.Frame
		if !testCase.ExpectSilence {
			expected, err = socketcan.ParseFrame(testCase.Expected)
			if err != nil {
				return nil, fmt.Errorf("case %q expected response: %w", testCase.Name, err)
			}
		}

		timeout := testCase.Timeout.Duration
		if timeout <= 0 {
			timeout = time.Second
		}
		prepared = append(prepared, PreparedCase{
			Name:          testCase.Name,
			Request:       request,
			Expected:      expected,
			ExpectSilence: testCase.ExpectSilence,
			Timeout:       timeout,
		})
	}
	return prepared, nil
}
