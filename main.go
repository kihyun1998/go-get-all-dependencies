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
		"golang.org/x/tools":         "github.com/golang/tools",
		"golang.org/x/sync":          "github.com/golang/sync",
		"golang.org/x/text":          "github.com/golang/text",
		"golang.org/x/net":           "github.com/golang/net",
		"golang.org/x/sys":           "github.com/golang/sys",
		"golang.org/x/crypto":        "github.com/golang/crypto",
		"golang.org/x/mod":           "github.com/golang/mod",
		"golang.org/x/oauth2":        "github.com/golang/oauth2",
		"google.golang.org/protobuf": "github.com/protocolbuffers/protobuf-go",
		"google.golang.org/grpc":     "github.com/grpc/grpc-go",
		"google.golang.org/genproto": "github.com/googleapis/go-genproto",
		"k8s.io/api":                 "github.com/kubernetes/api",
		"k8s.io/client-go":           "github.com/kubernetes/client-go",
		"k8s.io/apimachinery":        "github.com/kubernetes/apimachinery",
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
	fmt.Println("\nreplace (")
	for _, dep := range da.uniqueDeps {
		replacement := da.getReplacement(dep)
		if replacement != "" {
			fmt.Printf("\t%s => %s\n", dep.FullPath, replacement)
		}
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

	// 도메인별로 정렬하여 출력
	for domain, paths := range pathsByDomain {
		fmt.Printf("\n%s:\n", domain)
		// paths 정렬
		sort.Strings(paths)
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
