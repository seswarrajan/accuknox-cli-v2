package imagescan

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/accuknox/kubeshield/api/v1beta1"
	kubesheildDiscovery "github.com/accuknox/kubeshield/pkg/discovery"
	httpclient "github.com/accuknox/kubeshield/pkg/scanner/httpClient"
	kubesheildScanner "github.com/accuknox/kubeshield/pkg/scanner/scan"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

// Discovers the running container images and scans the images using the specified tool
func DiscoverAndScan(conf kubesheildScanner.ScanConfig, hostName, runtime string, onlyRunningContainers, onlyImages bool) error {
	zapLogger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("failed to initialize logger")
	}

	defer func() {
		// Ignoring EINVAL errors based on https://github.com/uber-go/zap/issues/328#issuecomment-284337436
		if err := zapLogger.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) {
			fmt.Printf("error: %v\n", err)
		}
	}()

	// Install trivy if it is not exists
	if !IsTrivyInstalled() {
		if err := installTrivy(); err != nil {
			return fmt.Errorf("error while installing container image scanner: %v", err)
		}
		zapLogger.Info("Dowloaded container image scanner successfully")
		// Remove trivy binary, if it is installed by knoxctl
		defer cleanupInstalledBinaryPath()
	}

	conf.Images = discoverImages(hostName, runtime, onlyRunningContainers, onlyImages, zapLogger.Sugar())
	if len(conf.Images) == 0 {
		return fmt.Errorf("no images found for scanning")
	}

	// removes duplicate images
	conf.Images = lo.UniqBy(conf.Images, func(img v1beta1.Image) string {
		return img.Name
	})

	for i := range conf.Images {
		zapLogger.Sugar().Infof("Image Name: %s | Runtime: %s", conf.Images[i].Name, conf.Images[i].Runtime)
	}

	zapLogger.Info("Images Discovered Successfully", zap.Int("Total number of images:", len(conf.Images)))

	if hostName == "" {
		hostName, _ = os.Hostname()
	}

	// Additional fields added along with the scan results while calling artifact API
	conf.ArtifactConfig.AdditionalData = map[string]any{"host_name": hostName}
	conf.ScanTool = "trivy" // Default scanning tool

	imageScanner := kubesheildScanner.New(conf)
	imageScanner.ScannerHttpClient = httpclient.New()

	// Scans the provided images and sends the result back to saas through the artifact API
	if err := imageScanner.Scan(); err != nil {
		return fmt.Errorf("error while scanning the images")
	}

	zapLogger.Info("Images Scanned Successfully",
		zap.Int("Total Scanned Images", len(conf.Images)),
		zap.String("Tool used for scanning", conf.ScanTool))

	return nil
}

// Lists the running containers for the provided runtime, if the runtime is empty it will use the default supported runtimes
func discoverImages(hostName, runtime string, onlyRunningContainers, imageOnly bool, logger *zap.SugaredLogger) []v1beta1.Image {
	var (
		runtimes = []string{"docker", "containerd", "cri-o", "nri"}
		images   []v1beta1.Image
	)

	if runtime != "" {
		runtimes = []string{runtime}
	}

	// Fetching images present in all the provided runtimes
	for _, r := range runtimes {
		detectedRuntime, criPath, ok := kubesheildDiscovery.DiscoverNodeRuntime("", r, logger)
		if !ok {
			logger.Errorf("Unable to detect runtime for %s", r)
			continue
		}

		// If imageOnly flag is enabled, we only discover images; not containers
		if imageOnly {
			imageList, err := kubesheildDiscovery.ListImages(detectedRuntime, criPath, kubesheildDiscovery.VM)
			if err != nil {
				logger.Errorf("error while listing the images for runtime %s: %v", r, err)
				continue
			}
			images = append(images, imageList...)
			continue
		}

		// By default we fetch running containers, unless onlyRunningContainers is set to false
		containerList, err := kubesheildDiscovery.ListContainers(detectedRuntime, criPath, kubesheildDiscovery.VM, onlyRunningContainers)
		if err != nil {
			logger.Errorf("error while listing the container images for runtime %s: %v", r, err)
			continue
		}
		images = append(images, containerList...)
	}
	return images
}
