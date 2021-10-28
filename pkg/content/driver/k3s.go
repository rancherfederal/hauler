package driver

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rancherfederal/hauler/pkg/apis/hauler.cattle.io/v1alpha1"
	"github.com/rancherfederal/hauler/pkg/content/file"
	"github.com/rancherfederal/hauler/pkg/content/image"
)

type K3s struct {
	Files  []file.File
	Images []image.Image
}

func NewK3s(version string) (*K3s, error) {
	bom, err := newBom("k3s", version)
	if err != nil {
		return nil, err
	}

	var files []file.File
	for _, f := range bom.files.Spec.Files {
		fi := file.NewFile(f)
		files = append(files, fi)
	}

	var images []image.Image
	for _, i := range bom.images.Spec.Images {
		img := image.NewImage(i)
		images = append(images, img)
	}

	return &K3s{
		Files:  files,
		Images: images,
	}, nil
}

func (k *K3s) Copy(ctx context.Context, registry string) error {
	for _, f := range k.Files {
		if err := f.Copy(ctx, registry); err != nil {
			return err
		}
	}

	for _, i := range k.Images {
		if err := i.Copy(ctx, registry); err != nil {
			return err
		}
	}

	return nil
}

type bom struct {
	files  v1alpha1.Files
	images v1alpha1.Images
}

// TODO: support multi-arch with variadic options
func newBom(kind string, version string) (bom, error) {
	files := v1alpha1.Files{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.String(),
			Kind:       v1alpha1.FilesContentKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: kind,
		},
		Spec: v1alpha1.FileSpec{
			Files: []v1alpha1.File{},
		},
	}

	images := v1alpha1.Images{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.String(),
			Kind:       v1alpha1.ImagesContentKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: kind,
		},
		Spec: v1alpha1.ImageSpec{
			Images: []v1alpha1.Image{},
		},
	}

	var imgs, fls []string = nil, nil

	switch kind {
	case "rke2":
		releaseUrl := "https://github.com/rancher/rke2/releases/download"
		u, _ := url.Parse(releaseUrl)
		u.Path = path.Join(u.Path, version, "rke2-images-all.linux-amd64.txt")

		rke2Images, err := parseImageList(u.String())
		if err != nil {
			return bom{}, err
		}

		imgs = rke2Images

		fls = []string{
			path.Join(releaseUrl, version, "rke2.linux-amd64"),
		}

	case "k3s":
		r := release("https://github.com/k3s-io/k3s/releases/download")

		k3sImages, err := parseImageList(r.Join(version, "k3s-images.txt"))
		if err != nil {
			return bom{}, err
		}

		imgs = k3sImages

		fls = []string{
			r.Join(version, "k3s"),
		}

	default:
		return bom{}, fmt.Errorf("%s is not a valid driver kind", kind)
	}

	for _, fi := range fls {
		f := v1alpha1.File{
			Ref: fi,
		}

		files.Spec.Files = append(files.Spec.Files, f)
	}

	for _, i := range imgs {
		img := v1alpha1.Image{Ref: i}
		images.Spec.Images = append(images.Spec.Images, img)
	}

	return bom{
		files:  files,
		images: images,
	}, nil
}

// parseImageList is a helper function to fetch and parse k3s/rke2 release image lists
func parseImageList(ref string) ([]string, error) {
	resp, err := http.Get(ref)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var imgs []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		imgs = append(imgs, scanner.Text())
	}

	return imgs, nil
}

type release string

func (r release) Join(component ...string) string {
	u, _ := url.Parse(string(r))
	complete := []string{u.Path}
	u.Path = path.Join(append(complete, component...)...)
	return u.String()
}
