package main

import (
	"strconv"
	"testing"
	"time"
)

func TestExtractTitlePresentLowerCase(t *testing.T) {
	body := "this is a dummy<title>test page</title> to make sure title extracting works great"
	title := extractTitle(body)

	if title != "test page" {
		t.Errorf("Found title: %v is not 'test page'", title)
	}
}

func TestExtractTitlePresentUpperCaseAndPreserveCase(t *testing.T) {
	body := "this is A dummy<TITLE>test PaGe</TITLE> to MAKE sure title EXTRACTING works great"
	title := extractTitle(body)

	if title != "test PaGe" {
		t.Errorf("Found title: %v is not 'test PaGe'", title)
	}
}

func TestExtractTitleNotPresent(t *testing.T) {
	body := "this is a dummy to make sure title extracting works great"
	title := extractTitle(body)

	if title != "" {
		t.Errorf("Found title but no one are present")
	}
}

func TestComputePathNoExtension(t *testing.T) {
	currentTime := time.Now()
	path, fileName := computePath("https://private.creekorful.fr/something/strange/", currentTime)

	if path != "private.creekorful.fr/something/strange" || fileName != strconv.FormatInt(currentTime.Unix(), 10) {
		t.Errorf("Computed path is wrong")
	}
}

func TestComputePathWithExtension(t *testing.T) {
	currentTime := time.Now()
	path, fileName := computePath("http://private.creekorful.fr/something/strange/index.php", currentTime)

	if path != "private.creekorful.fr/something/strange/index.php" || fileName != strconv.FormatInt(currentTime.Unix(), 10) {
		t.Errorf("Computed path is wrong")
	}
}
