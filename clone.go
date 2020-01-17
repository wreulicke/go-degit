package main

import (
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/src-d/go-billy.v4/memfs"

	"github.com/sirupsen/logrus"
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

type Copier struct {
	logger *logrus.Logger
	prefix string
	dest   string
}

func (c *Copier) WriteResource(fs billy.Filesystem) error {
	d, err := fs.ReadDir("/")
	if err != nil {
		return err
	}
	return c.visitFiles(fs, d, []string{})
}

func (c *Copier) visitFiles(fs billy.Filesystem, files []os.FileInfo, base []string) error {
	for _, e := range files {
		err := c.visitFile(fs, e, base)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Copier) visitDirectory(fs billy.Filesystem, f os.FileInfo, base []string) error {
	base = append(base, f.Name())
	p := path.Join(base...)
	d, err := fs.ReadDir(p)
	if err != nil {
		return err
	}
	return c.visitFiles(fs, d, base)
}

func (c *Copier) copyFile(fs billy.Filesystem, f os.FileInfo, base []string) error {
	base = append(base, f.Name())
	p := path.Join(base...)
	if !strings.HasPrefix(p, c.prefix) {
		return nil
	}
	c.logger.Infof("%s %s", filepath.Join(c.dest, filepath.Join(base...)), c.prefix)
	destPath := filepath.Join(c.dest, strings.TrimPrefix(filepath.Join(base...), c.prefix))
	basePath := filepath.Dir(destPath)
	c.logger.Infof("matched! copy %s to %s", p, destPath)
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

func (c *Copier) visitFile(fs billy.Filesystem, f os.FileInfo, base []string) error {
	if f.IsDir() {
		return c.visitDirectory(fs, f, base)
	}
	return c.copyFile(fs, f, base)
}

// Clone repository into destination striping prefix
func Clone(l *logrus.Logger, u *url.URL, prefix string, dest string) error {
	l.Infof("Cloning... %s into %s", u.String(), dest)
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
	cp := Copier{
		dest:   dest,
		prefix: prefix,
		logger: l,
	}
	return cp.WriteResource(f)
}

func mkdirRecursively(path string) error {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return nil
	}
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
