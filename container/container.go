package container

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"mydocker/constant"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
)

// NewParentProcess 启动一个新进程
/*
这里是父进程，也就是当前进程执行的内容。
1.这里的/proc/se1f/exe调用中，/proc/self/ 指的是当前运行进程自己的环境，exec 其实就是自己调用了自己，使用这种方式对创建出来的进程进行初始化
2.后面的args是参数，其中init是传递给本进程的第一个参数，在本例中，其实就是会去调用initCommand去初始化进程的一些环境和资源
3.下面的clone参数就是去fork出来一个新进程，并且使用了namespace隔离新创建的进程和外部环境。
4.如果用户指定了-it参数，就需要把当前进程的输入输出导入到标准输入输出上
*/
func NewParentProcess(tty bool, volume string) (*exec.Cmd, *os.File) {
	// 创建匿名管道用于传递参数，将readPipe作为子进程的ExtraFiles，子进程从readPipe中读取参数
	// 父进程中则通过writePipe将参数写入管道
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		log.Errorf("New pipe error %v", err)
		return nil, nil
	}

	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	cmd.ExtraFiles = []*os.File{readPipe}
	//mntURL := "/home/win233/mydocker/merged/"
	rootPath := "/home/win233/mydocker/"
	NewWorkSpace(rootPath, volume)
	cmd.Dir = path.Join(rootPath, "merged")
	return cmd, writePipe
}

// NewWorkSpace Create an Overlay2 filesystem as container root workspace
func NewWorkSpace(rootPath string, volumePath string) {
	createLower(rootPath)
	createDirs(rootPath)
	mountOverlayFS(rootPath)

	//处理volume
	if volumePath != "" {
		mntPath := path.Join(rootPath, "merged")
		hostPath, containerPath, err := volumeExtract(volumePath)
		if err != nil {
			log.Errorf("extract volume failed，maybe volume parameter input is not correct，detail:%v", err)
			return
		}
		mountVolume(mntPath, hostPath, containerPath)
	}

}

// mountVolume 使用 bind mount 挂载 volume
func mountVolume(mntPath, hostPath, containerPath string) {
	// 创建宿主机目录
	if err := os.Mkdir(hostPath, constant.Perm0777); err != nil {
		log.Infof("mkdir parent dir %s error. %v", hostPath, err)
	}
	// 拼接出对应的容器目录在宿主机上的的位置，并创建对应目录
	containerPathInHost := path.Join(mntPath, containerPath)
	if err := os.Mkdir(containerPathInHost, constant.Perm0777); err != nil {
		log.Infof("mkdir container dir %s error. %v", containerPathInHost, err)
	}
	// 通过bind mount 将宿主机目录挂载到容器目录
	// mount -o bind /hostPath /containerPath
	cmd := exec.Command("mount", "-o", "bind", hostPath, containerPathInHost)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("mount volume failed. %v", err)
	}
}

// volumeExtract 通过冒号分割解析volume目录，比如 -v /tmp:/tmp
func volumeExtract(volume string) (sourcePath, destinationPath string, err error) {
	parts := strings.Split(volume, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid volume [%s], must split by `:`", volume)
	}

	sourcePath, destinationPath = parts[0], parts[1]
	if sourcePath == "" || destinationPath == "" {
		return "", "", fmt.Errorf("invalid volume [%s], path can't be empty", volume)
	}

	return sourcePath, destinationPath, nil
}

// createLower 将busybox作为overlayfs的lower层
func createLower(rootURL string) {
	// 把busybox作为overlayfs中的lower层
	busyboxURL := rootURL + "busybox/"
	busyboxTarURL := rootURL + "busybox.tar"
	// 检查是否已经存在busybox文件夹
	exist, err := PathExists(busyboxURL)
	if err != nil {
		log.Infof("Fail to judge whether dir %s exists. %v", busyboxURL, err)
	}
	// 不存在则创建目录并将busybox.tar解压到busybox文件夹中
	if !exist {
		if err := os.Mkdir(busyboxURL, 0777); err != nil {
			log.Errorf("Mkdir dir %s error. %v", busyboxURL, err)
		}
		log.Infoln("start extract files...")
		if _, err := exec.Command("tar", "-xvf", busyboxTarURL, "-C", busyboxURL).CombinedOutput(); err != nil {
			log.Errorf("Untar dir %s error %v", busyboxURL, err)
		}
	}
}

// createDirs 创建overlayfs需要的的upper、worker目录
func createDirs(rootPath string) {
	dirs := []string{
		path.Join(rootPath, "merged"),
		path.Join(rootPath, "upper"),
		path.Join(rootPath, "work"),
	}

	for _, dir := range dirs {
		if err := os.Mkdir(dir, 0777); err != nil {
			log.Errorf("mkdir dir %s error. %v", dir, err)
		}
	}
}

// mountOverlayFS 挂载overlayfs
func mountOverlayFS(rootPath string) {
	// 拼接参数
	// e.g. lowerdir=/root/busybox,upperdir=/root/upper,workdir=/root/work
	dirs := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", path.Join(rootPath, "busybox"),
		path.Join(rootPath, "upper"), path.Join(rootPath, "work"))

	// 完整命令：mount -t overlay overlay -o lowerdir=/root/busybox,upperdir=/root/upper,workdir=/root/work /root/merged
	cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", dirs, path.Join(rootPath, "merged"))
	log.Infof("mount overlayfs: [%s]", cmd.String())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("%v", err)
	}
}

// DeleteWorkSpace Delete the AUFS filesystem while container exit
func DeleteWorkSpace(rootPath string, volumePath string) {
	mntPath := path.Join(rootPath, "merged")

	// 如果指定了volume则需要umount volume
	// NOTE: 一定要要先 umount volume ，然后再删除目录，否则由于 bind mount 存在，删除临时目录会导致 volume 目录中的数据丢失。
	if volumePath != "" {
		_, containerPath, err := volumeExtract(volumePath)
		if err != nil {
			log.Errorf("extract volume failed，maybe volume parameter input is not correct，detail:%v", err)
			return
		}
		umountVolume(mntPath, containerPath)
	}

	umountOverlayFS(path.Join(rootPath, "merged"))
	deleteDirs(rootPath)
}

func umountVolume(mntPath, containerPath string) {
	// mntPath 为容器在宿主机上的挂载点，例如 /root/merged
	// containerPath 为 volume 在容器中对应的目录，例如 /root/tmp
	// containerPathInHost 则是容器中目录在宿主机上的具体位置，例如 /root/merged/root/tmp
	containerPathInHost := path.Join(mntPath, containerPath)
	cmd := exec.Command("umount", containerPathInHost)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("Umount volume failed. %v", err)
	}
}

func umountOverlayFS(mntURL string) {
	cmd := exec.Command("umount", mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("%v", err)
	}
	if err := os.RemoveAll(mntURL); err != nil {
		log.Errorf("Remove dir %s error %v", mntURL, err)
	}
}

func deleteDirs(rootPath string) {
	dirs := []string{
		path.Join(rootPath, "merged"),
		path.Join(rootPath, "upper"),
		path.Join(rootPath, "work"),
	}

	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			log.Errorf("Remove dir %s error %v", dir, err)
		}
	}
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
