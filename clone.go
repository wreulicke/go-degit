package main

import (
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/src-d/go-billy.v4/memfs"

	"fmt"
	"log"

	giturls "github.com/whilp/git-urls"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

func toURL(rawurl string) (*url.URL, error) {
	if strings.HasPrefix(rawurl, "git@") {
		if strings.HasSuffix(rawurl, ".git") {
			return giturls.Parse(rawurl)
		}
		return giturls.Parse(rawurl + ".git")
	} else if strings.HasPrefix(rawurl, "https://") {
		if strings.HasSuffix(rawurl, ".git") {
			return url.Parse(rawurl)
		}
		return url.Parse(rawurl + ".git")
	}
	return url.Parse("https://" + path.Join("github.com", rawurl+".git"))
}

type PrefixMatcher struct {
	prefix string
}

func (p *PrefixMatcher) Match(path string) bool {
	return strings.HasPrefix(path, p.prefix)
}

func List(fs billy.Filesystem, f os.FileInfo, base string) {
	if f.IsDir() {
		d, err := fs.ReadDir(path.Join(base, f.Name()))
		if err != nil {
			return
		}
		for _, e := range d {
			List(fs, e, path.Join(base, f.Name()))
		}
		return
	}
	fmt.Println(path.Join(base, f.Name()))
}

func Clone(u *url.URL, prefix string, dest string) error {
	log.Printf("Cloning... %s into %s\n", u, dest)
	f := memfs.New()
	c := git.CloneOptions{
		URL:           u.String(),
		ReferenceName: plumbing.ReferenceName("refs/heads/master"),
		Depth:         1,
	}
	_, err := git.Clone(memory.NewStorage(), f, &c)
	if err != nil {
		return err
	}
	m := &PrefixMatcher{
		prefix: prefix,
	}
	return WriteResource(f, dest, m)
}

func mkdirRecursively(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		err := os.Mkdir(path, 0771)
		if _, ok := err.(*os.PathError); ok {
			err := mkdirRecursively(filepath.Dir(path))
			if err != nil {
				return err
			}
			if err := os.Mkdir(path, 0771); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	return nil
}

func WriteResource(fs billy.Filesystem, dest string, m *PrefixMatcher) error {
	d, _ := fs.ReadDir("/")
	for _, e := range d {
		err := writeResourceInternal(fs, e, dest, m, []string{})
		if err != nil {
			return err
		}
	}
	return nil
}

func writeResourceInternal(fs billy.Filesystem, f os.FileInfo, dest string, m *PrefixMatcher, base []string) error {
	base = append(base, f.Name())
	p := path.Join(base...)
	if f.IsDir() {
		d, err := fs.ReadDir(p)
		if err != nil {
			return err
		}
		for _, e := range d {
			err := writeResourceInternal(fs, e, dest, m, base)
			if err != nil {
				return err
			}
		}
		return nil
	}
	destPath := filepath.Join(dest, filepath.Join(base...))
	if !m.Match(p) {
		return nil
	}
	basePath := filepath.Dir(destPath)
	err := mkdirRecursively(basePath)
	if err != nil {
		return err
	}
	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE, f.Mode().Perm())
	if err != nil {
		return err
	}
	file, err := fs.Open(p)
	if err != nil {
		return err
	}
	_, err = io.Copy(destFile, file)
	return err
}
