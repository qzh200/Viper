//go:build windows

package wmi

import (
	"testing"
)

// 一个方法的输出参数是按名字读的。提供程序的名字是主路，输出参数定义的
// 属性表只是一条补充的路。

func TestTheConventionalOutputNamesCarryTheNamesOfTheCpu(t *testing.T) {
	// 超频的方法给出的三个名字不是平常那些。
	names := map[string]bool{}
	for _, name := range conventionalOutputNames {
		names[name] = true
	}
	for _, want := range []string{
		"gmOutput", "uiOutput", "gmReturn",
		"ReturnCode", "ReturnOCProfile", "OCStructure",
	} {
		if !names[want] {
			t.Errorf("名字 %q 没有被试，提供程序的名字是 %v", want, conventionalOutputNames)
		}
	}
}

func TestTheOutputNamesAreMergedWithThePropertyList(t *testing.T) {
	// 属性表里的名字也一并被试，这样一个给输出参数起别的名字的提供程序
	// 不会被漏掉。
	merged := mergeOutputNames([]string{"ReturnCode", "SomethingElse", ""})

	if len(merged) != len(conventionalOutputNames)+1 {
		t.Fatalf("这些名字是 %v, 要提供程序的名字再加一个", merged)
	}
	if merged[len(merged)-1] != "SomethingElse" {
		t.Errorf("最后一个名字是 %q, 要 SomethingElse", merged[len(merged)-1])
	}
	// 提供程序也用的那个名字不会出两遍。
	seen := map[string]bool{}
	for _, name := range merged {
		if seen[name] {
			t.Errorf("名字 %q 出了两遍", name)
		}
		seen[name] = true
	}
}

func TestTheOutputNamesOfAListThatCannotBeRead(t *testing.T) {
	// 没有输出参数的定义，属性表就读不出来，所以只剩提供程序的名字。
	names := outputNames(0)
	if len(names) != len(conventionalOutputNames) {
		t.Fatalf("这些名字是 %v, 要提供程序的名字", names)
	}
	for i, want := range conventionalOutputNames {
		if names[i] != want {
			t.Errorf("第 %d 个名字是 %q, 要 %q", i, names[i], want)
		}
	}
}

func TestTheOutputNamesOfAListThatIsRead(t *testing.T) {
	// 提供程序的名字排在前面，所以它们比属性表里的名字先被试。
	merged := mergeOutputNames([]string{"SomethingElse"})
	if merged[0] != conventionalOutputNames[0] {
		t.Errorf("第一个名字是 %q, 要 %q", merged[0], conventionalOutputNames[0])
	}
}

func TestContains(t *testing.T) {
	names := []string{"a", "b"}
	if !contains(names, "a") {
		t.Error("名字 a 没有被找到")
	}
	if contains(names, "c") {
		t.Error("名字 c 被找到了")
	}
}
