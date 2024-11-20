package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
)

type Dependency struct {
	FullPath string
	Name     string
	Version  string
}

type DependencyAnalyzer struct {
	dependencies []*Dependency
	uniqueDeps   map[string]*Dependency
	replaceRules map[string]string
	unknownPaths map[string]bool // unknownDomains를 unknownPaths로 변경
	filePath     string
}

func NewDependencyAnalyzer(filePath string) *DependencyAnalyzer {
	// 기본 replace 규칙 설정 - 정확한 경로만 매칭
	defaultRules := map[string]string{
		// golang
		"golang.org/x/tools":        "github.com/golang/tools",
		"golang.org/x/sync":         "github.com/golang/sync",
		"golang.org/x/text":         "github.com/golang/text",
		"golang.org/x/net":          "github.com/golang/net",
		"golang.org/x/sys":          "github.com/golang/sys",
		"golang.org/x/crypto":       "github.com/golang/crypto",
		"golang.org/x/mod":          "github.com/golang/mod",
		"golang.org/x/oauth2":       "github.com/golang/oauth2",
		"golang.org/x/exp":          "github.com/golang/exp",
		"golang.org/x/exp/shiny":    "github.com/golang/exp/shiny",
		"golang.org/x/image":        "github.com/golang/image",
		"golang.org/x/lint":         "github.com/golang/lint",
		"golang.org/x/mobile":       "github.com/golang/mobile",
		"golang.org/x/term":         "github.com/golang/term",
		"golang.org/x/time":         "github.com/golang/time",
		"golang.org/x/xerrors":      "github.com/golang/xerrors",
		"golang.org/x/tools/go/vcs": "github.com/golang/tools/go/vcs",

		// google golang
		"google.golang.org/api":       "github.com/googleapis/google-api-go-client",
		"google.golang.org/appengine": "github.com/golang/appengine",
		"google.golang.org/protobuf":  "github.com/protocolbuffers/protobuf-go",
		"google.golang.org/grpc":      "github.com/grpc/grpc-go",
		"google.golang.org/genproto":  "github.com/googleapis/go-genproto",

		// go pkg
		"gopkg.in/check.v1": "github.com/go-check/check/tree/v1",
		"gopkg.in/errgo.v2": "github.com/go-errgo/errgo/tree/v2.1.0",
		"gopkg.in/ini.v1":   "github.com/go-ini/ini/tree/v1.67.0",
		"gopkg.in/yaml.v2":  "github.com/go-yaml/yaml/tree/v2.4.0",
		"gopkg.in/yaml.v3":  "github.com/go-yaml/yaml/tree/v3.0.1",

		"honnef.co/go/js/dom": "github.com/dominikh/go-js-dom",
		"honnef.co/go/tools":  "github.com/dominikh/go-tools",

		"rsc.io/binaryregexp": "github.com/rsc/binaryregexp",
		"rsc.io/quote/v3":     "github.com/rsc/quote/v3",
		"rsc.io/sampler":      "github.com/rsc/sampler",

		// cloud
		"cloud.google.com/go":           "github.com/googleapis/google-cloud-go",
		"cloud.google.com/go/bigquery":  "github.com/googleapis/google-cloud-go/bigquery",
		"cloud.google.com/go/datastore": "github.com/googleapis/google-cloud-go/datastore",
		"cloud.google.com/go/firestore": "github.com/googleapis/google-cloud-go/firestore",
		"cloud.google.com/go/pubsub":    "github.com/googleapis/google-cloud-go/pubsub",
		"cloud.google.com/go/storage":   "github.com/googleapis/google-cloud-go/storage",
		// fyne
		"fyne.io/fyne/v2": "github.com/fyne-io/fyne/v2",
		"fyne.io/systray": "github.com/fyne-io/systray",
		// go etcd
		"go.etcd.io/etcd/api/v3":        "github.com/etcd-io/etcd/api/v3",
		"go.etcd.io/etcd/client/pkg/v3": "github.com/etcd-io/etcd/client/pkg/v3",
		"go.etcd.io/etcd/client/v2":     "github.com/etcd-io/etcd/client/v2",
		// go opencensus
		"go.opencensus.io": "github.com/census-instrumentation/opencensus-go",
		// uber
		"go.uber.org/atomic":   "github.com/uber-go/atomic",
		"go.uber.org/multierr": "github.com/uber-go/multierr",
		"go.uber.org/zap":      "github.com/uber-go/zap",
		// "k8s.io/api":                 "github.com/kubernetes/api",
		// "k8s.io/client-go":           "github.com/kubernetes/client-go",
		// "k8s.io/apimachinery":        "github.com/kubernetes/apimachinery",
	}

	return &DependencyAnalyzer{
		dependencies: make([]*Dependency, 0),
		uniqueDeps:   make(map[string]*Dependency),
		replaceRules: defaultRules,
		unknownPaths: make(map[string]bool),
		filePath:     filePath,
	}
}

func (da *DependencyAnalyzer) getReplacement(dep *Dependency) string {
	if strings.HasPrefix(dep.Name, "github.com/") {
		return ""
	}

	if to, exists := da.replaceRules[dep.Name]; exists {
		return fmt.Sprintf("%s %s", to, dep.Version)
	}

	da.unknownPaths[dep.Name] = true
	return ""
}

func parseDependency(dep string) *Dependency {
	// @ 기호가 있으면 공백으로 대체
	dep = strings.Replace(dep, "@", " ", 1)

	parts := strings.Fields(dep)
	if len(parts) != 2 {
		return &Dependency{FullPath: dep, Name: dep}
	}
	return &Dependency{
		FullPath: dep,
		Name:     parts[0],
		Version:  parts[1],
	}
}

func (da *DependencyAnalyzer) Analyze() error {
	var reader *bufio.Reader

	if da.filePath == "-" {
		reader = bufio.NewReader(os.Stdin)
	} else {
		file, err := os.Open(da.filePath)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()
		reader = bufio.NewReader(file)
	}

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)

		if len(parts) > 1 {
			dep := parseDependency(parts[1])
			da.dependencies = append(da.dependencies, dep)
			key := fmt.Sprintf("%s@%s", dep.Name, dep.Version)
			da.uniqueDeps[key] = dep
		} else if len(parts) == 1 {
			dep := parseDependency(parts[0])
			da.dependencies = append(da.dependencies, dep)
			key := fmt.Sprintf("%s@%s", dep.Name, dep.Version)
			da.uniqueDeps[key] = dep
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	return nil
}

func (da *DependencyAnalyzer) PrintDependencies() {
	// replace 규칙을 정렬하기 위해 키들을 슬라이스로 추출 후 정렬
	var replacements []string
	for _, dep := range da.uniqueDeps {
		replacement := da.getReplacement(dep)
		if replacement != "" {
			// 전체 replace 문장을 슬라이스에 저장
			replacements = append(replacements,
				fmt.Sprintf("\t%s => %s", dep.FullPath, replacement))
		}
	}
	// 정렬
	sort.Strings(replacements)

	fmt.Println("\nreplace (")
	// 정렬된 순서로 출력
	for _, r := range replacements {
		fmt.Println(r)
	}
	fmt.Println(")")

	fmt.Println("\nUnhandled Paths:")
	fmt.Println("===============")

	// 경로를 도메인별로 그룹화
	pathsByDomain := make(map[string][]string)
	for path := range da.unknownPaths {
		if !strings.HasPrefix(path, "github.com") {
			parts := strings.SplitN(path, "/", 2)
			domain := parts[0]
			pathsByDomain[domain] = append(pathsByDomain[domain], path)
		}
	}

	// 도메인들을 슬라이스로 추출하여 정렬
	var domains []string
	for domain := range pathsByDomain {
		domains = append(domains, domain)
	}
	sort.Strings(domains)

	// 정렬된 도메인 순서로 출력
	for _, domain := range domains {
		fmt.Printf("\n%s:\n", domain)
		paths := pathsByDomain[domain]
		sort.Strings(paths) // 각 도메인의 경로도 정렬
		for _, path := range paths {
			fmt.Printf("  %s\n", path)
		}
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: program <dependency-file>")
		fmt.Println("       program - (read from stdin)")
		os.Exit(1)
	}

	analyzer := NewDependencyAnalyzer(os.Args[1])
	if err := analyzer.Analyze(); err != nil {
		fmt.Printf("Error analyzing dependencies: %v\n", err)
		os.Exit(1)
	}

	analyzer.PrintDependencies()
}
