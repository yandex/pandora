package acceptance

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/yandex/pandora/lib/ginkgoutil"
	"github.com/yandex/pandora/lib/tag"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

var pandoraBin string

func TestAcceptanceTests(t *testing.T) {
	ginkgoutil.SetupSuite()
	var args []string
	if tag.Race {
		zap.L().Debug("Building with race detector")
		args = append(args, "-race")
	}
	if tag.Debug {
		zap.L().Debug("Building with debug tag")
		args = append(args, "-tags", "debug")
	}
	var err error
	pandoraBin, err = gexec.Build("github.com/yandex/pandora", args...)
	if err != nil {
		t.Fatal(err)
	}
	defer gexec.CleanupBuildArtifacts()
	RunSpecs(t, "AcceptanceTests Suite")
}

type TestConfig struct {
	// Default way to pass config to pandora.
	PandoraConfig
	// RawConfig overrides Pandora.
	RawConfig  string
	ConfigName string            // Without extension. "load" by default.
	UseJSON    bool              // Using YAML by default.
	CmdArgs    []string          // Nothing by default.
	Files      map[string]string // Extra files to put in dir. Ammo, etc.
}

func NewTestConfig() *TestConfig {
	return &TestConfig{
		PandoraConfig: PandoraConfig{
			Pool: []*InstancePoolConfig{NewInstansePoolConfig()},
		},
		Files: map[string]string{},
	}
}

type PandoraConfig struct {
	Pool             []*InstancePoolConfig `yaml:"pools" json:"pools"`
	LogConfig        `yaml:"log,omitempty" json:"log,omitempty"`
	MonitoringConfig `yaml:"monitoring,omitempty" json:"monitoring,omitempty"`
}

type LogConfig struct {
	Level string `yaml:"level,omitempty" json:"level,omitempty"`
	File  string `yaml:"file,omitempty" json:"file,omitempty"`
}

type MonitoringConfig struct {
	Expvar     *expvarConfig     `yaml:"Expvar"`
	CPUProfile *cpuprofileConfig `yaml:"CPUProfile"`
	MemProfile *memprofileConfig `yaml:"MemProfile"`
}

type expvarConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
	Port    int  `yaml:"port" json:"port"`
}

type cpuprofileConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	File    string `yaml:"file" json:"file"`
}

type memprofileConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	File    string `yaml:"file" json:"file"`
}

func (pc *PandoraConfig) Append(ipc *InstancePoolConfig) {
	pc.Pool = append(pc.Pool, ipc)
}

func NewInstansePoolConfig() *InstancePoolConfig {
	return &InstancePoolConfig{
		Provider:        map[string]interface{}{},
		Aggregator:      map[string]interface{}{},
		Gun:             map[string]interface{}{},
		RPSSchedule:     map[string]interface{}{},
		StartupSchedule: map[string]interface{}{},
	}

}

type InstancePoolConfig struct {
	ID              string
	Provider        map[string]interface{} `yaml:"ammo" json:"ammo"`
	Aggregator      map[string]interface{} `yaml:"result" json:"result"`
	Gun             map[string]interface{} `yaml:"gun" json:"gun"`
	RPSPerInstance  bool                   `yaml:"rps-per-instance" json:"rps-per-instance"`
	RPSSchedule     interface{}            `yaml:"rps" json:"rps"`
	StartupSchedule interface{}            `yaml:"startup" json:"startup"`
}

type PandoraTester struct {
	*gexec.Session
	// TestDir is working dir of launched pandora.
	// It contains config and ammo files, and will be removed after test execution.
	// All files created during a test should created in this dir.
	TestDir string
	Config  *TestConfig
}

func NewTester(conf *TestConfig) *PandoraTester {
	testDir, err := ioutil.TempDir("", "pandora_acceptance_")
	Expect(err).ToNot(HaveOccurred())
	if conf.ConfigName == "" {
		conf.ConfigName = "load"
	}
	extension := "yaml"
	if conf.UseJSON {
		extension = "json"
	}
	var confData []byte

	if conf.RawConfig != "" {
		confData = []byte(conf.RawConfig)
	} else {
		if conf.UseJSON {
			confData, err = json.Marshal(conf.PandoraConfig)
		} else {
			confData, err = yaml.Marshal(conf.PandoraConfig)
		}
		Expect(err).ToNot(HaveOccurred())
	}
	confAbsName := filepath.Join(testDir, conf.ConfigName+"."+extension)
	err = ioutil.WriteFile(confAbsName, confData, 0644)
	Expect(err).ToNot(HaveOccurred())

	for file, data := range conf.Files {
		fileAbsName := filepath.Join(testDir, file)
		err = ioutil.WriteFile(fileAbsName, []byte(data), 0644)
		Expect(err).ToNot(HaveOccurred())
	}

	command := exec.Command(pandoraBin, conf.CmdArgs...)
	command.Dir = testDir
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())

	tt := &PandoraTester{
		Session: session,
		TestDir: testDir,
		Config:  conf,
	}
	return tt
}

func (pt *PandoraTester) ShouldSay(pattern string) {
	EventuallyWithOffset(1, pt.Out, 3*time.Second).Should(gbytes.Say(pattern))
}

func (pt *PandoraTester) ExitCode() int {
	return pt.Session.Wait(5).ExitCode()
}

func (pt *PandoraTester) Close() {
	pt.Terminate()
	_ = os.RemoveAll(pt.TestDir)
}
