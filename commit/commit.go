package commit

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os/exec"
)

func CommitContainer(imageName string) error {
	mntPath := "/home/win233/mydocker/merged"
	imageTar := "/home/win233/mydocker/" + imageName + ".tar"
	fmt.Println("commitContainer imageTar:", imageTar)
	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntPath, ".").CombinedOutput(); err != nil {
		log.Errorf("tar folder %s error %v", mntPath, err)
		return err
	}
	return nil
}
