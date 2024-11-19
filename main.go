package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Dependency는 의존성 정보를 담는 구조체입니다
type Dependency struct {
	FullPath string // 전체 경로 (버전 포함)
	Name     string // 패키지 이름
	Version  string // 버전 정보
}

// DependencyAnalyzer는 의존성 분석을 위한 구조체입니다
type DependencyAnalyzer struct {
	dependencies map[string]*Dependency
	filePath     string
}

// NewDependencyAnalyzer creates a new analyzer
func NewDependencyAnalyzer(filePath string) *DependencyAnalyzer {
	return &DependencyAnalyzer{
		dependencies: make(map[string]*Dependency),
		filePath:     filePath,
	}
}

// parseDependency는 문자열을 Dependency 구조체로 파싱합니다
func parseDependency(dep string) *Dependency {
	parts := strings.Split(dep, "@")
	if len(parts) != 2 {
		return &Dependency{FullPath: dep, Name: dep}
	}
	return &Dependency{
		FullPath: dep,
		Name:     parts[0],
		Version:  parts[1],
	}
}

// Analyze 메서드는 파일을 읽고 의존성을 분석합니다
func (da *DependencyAnalyzer) Analyze() error {
	file, err := os.Open(da.filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)

		if len(parts) > 1 {
			// 두 번째 항목이 의존성
			dep := parseDependency(parts[1])
			da.dependencies[dep.Name] = dep
		} else if len(parts) == 1 {
			// 단일 항목인 경우도 의존성에 포함
			dep := parseDependency(parts[0])
			da.dependencies[dep.Name] = dep
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	return nil
}

// PrintDependencies 메서드는 수집된 의존성을 출력합니다
func (da *DependencyAnalyzer) PrintDependencies() {
	fmt.Println("Found Dependencies:")
	fmt.Println("==================")
	for _, dep := range da.dependencies {
		fmt.Printf("%s (%s)\n", dep.Name, dep.Version)
	}
	fmt.Printf("\nTotal dependencies: %d\n", len(da.dependencies))
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: program <dependency-file>")
		os.Exit(1)
	}

	analyzer := NewDependencyAnalyzer(os.Args[1])
	if err := analyzer.Analyze(); err != nil {
		fmt.Printf("Error analyzing dependencies: %v\n", err)
		os.Exit(1)
	}

	analyzer.PrintDependencies()
}
