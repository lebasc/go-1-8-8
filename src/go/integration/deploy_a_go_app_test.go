package integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CF Go Buildpack", func() {
	var app *cutlass.App
	var resource_url string

	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	Context("with cached buildpack dependencies", func() {
		BeforeEach(func() {
			if !cutlass.Cached {
				Skip("but running uncached tests")
			}
		})

		Context("app has dependencies", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_dependencies", "src", "with_dependencies"))
			})

			It("", func() {
				PushAppAndConfirm(app)
				Expect(app.Stdout.String()).To(MatchRegexp("Hello from foo!"))
				Expect(app.GetBody("/")).To(ContainSubstring("hello, world"))
			})

			AssertNoInternetTraffic("with_dependencies/src/with_dependencies")
		})

		Context("app uses go1.6 and godep with GO15VENDOREXPERIMENT=0", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "go16_dependencies"))
				app.SetEnv("GO15VENDOREXPERIMENT", "0")
			})

			It("", func() {
				PushAppAndConfirm(app)
				Expect(app.Stdout.String()).To(MatchRegexp("Hello from foo!"))
				Expect(app.GetBody("/")).To(ContainSubstring("hello, world"))
			})
		})

		Context("app uses go1.6 and godep with Godeps/_workspace dir", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "go16_dependencies"))
			})

			It("", func() {
				PushAppAndConfirm(app)
				Expect(app.Stdout.String()).To(MatchRegexp("Hello from foo!"))
				Expect(app.GetBody("/")).To(ContainSubstring("hello, world"))
			})
		})

		Context("app uses go1.6 with godep and no vendor dir or Godeps/_workspace dir", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "go16_no_vendor", "src", "go16_no_vendor"))
			})

			It("", func() {
				Expect(app.Push()).ToNot(BeNil())
				Expect(app.Stdout.String()).To(MatchRegexp("vendor/ directory does not exist."))
			})
		})

		Context("that are vendored", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "go17_vendor_experiment_flag"))
			})

			It("", func() {
				Expect(app.Push()).ToNot(BeNil())
				Expect(app.Stdout.String()).To(MatchRegexp("GO15VENDOREXPERIMENT is set, but is not supported by go1.7"))
			})
		})

		Context("app has vendored dependencies and no Godeps folder", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "native_vendoring"))
			})

			It("successfully stages", func() {
				PushAppAndConfirm(app)
				Expect(app.Stdout.String()).To(MatchRegexp("Init: a.A == 1"))
				Expect(app.GetBody("/")).To(ContainSubstring("Read: a.A == 1"))
			})

			AssertNoInternetTraffic("native_vendoring")
		})

		Context("app has vendored dependencies and custom package spec", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "vendored_custom_install_spec"))
				app.SetEnv("BP_DEBUG", "1")
			})

			It("successfully stages", func() {
				PushAppAndConfirm(app)
				Expect(app.Stdout.String()).To(MatchRegexp("Init: a.A == 1"))
				Expect(app.GetBody("/")).To(ContainSubstring("Read: a.A == 1"))
			})

			AssertNoInternetTraffic("vendored_custom_install_spec")
		})

		Context("app has vendored dependencies and a vendor.json file", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_vendor_json"))
			})

			It("successfully stages", func() {
				PushAppAndConfirm(app)
				Expect(app.Stdout.String()).To(MatchRegexp("Init: a.A == 1"))
				Expect(app.GetBody("/")).To(ContainSubstring("Read: a.A == 1"))
			})
		})

		Context("app with only a single go file and GOPACKAGENAME specified", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "single_file"))
			})

			It("successfully stages", func() {
				PushAppAndConfirm(app)
				Expect(app.GetBody("/")).To(ContainSubstring("simple apps are good"))
			})
		})

		Context("app with only a single go file, a vendor directory, and no GOPACKAGENAME specified", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "vendored_no_gopackagename"))
			})

			It("fails with helpful error", func() {
				Expect(app.Push()).ToNot(BeNil())
				Expect(app.Stdout.String()).To(MatchRegexp("failed"))
				Expect(app.Stdout.String()).To(MatchRegexp(`To use go native vendoring set the \$GOPACKAGENAME`))
			})
		})

		Context("app has vendored dependencies with go1.6, but GO15VENDOREXPERIMENT=0", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "go16_vendor_bad_env"))
			})

			It("fails with helpful error", func() {
				Expect(app.Push()).ToNot(BeNil())
				Expect(app.Stdout.String()).To(MatchRegexp("failed"))
				Expect(app.Stdout.String()).To(MatchRegexp("with go 1.6 this environment variable must unset or set to 1."))
			})
		})

		Context("app has no dependencies", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "go_app"))
			})

			It("", func() {
				PushAppAndConfirm(app)
				Expect(app.GetBody("/")).To(ContainSubstring("go, world"))
				Expect(app.Stdout.String()).To(MatchRegexp(`Installing go [\d\.]+`))
				Expect(app.Stdout.String()).To(MatchRegexp(`Copy \[\/tmp\/`))
			})
		})

		Context("app has before/after compile hooks", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "go_app"))
				app.SetEnv("BP_DEBUG", "1")
			})

			It("", func() {
				PushAppAndConfirm(app)
				Expect(app.GetBody("/")).To(ContainSubstring("go, world"))
				Expect(app.Stdout.String()).To(MatchRegexp("HOOKS 1: BeforeCompile"))
				Expect(app.Stdout.String()).To(MatchRegexp("HOOKS 2: AfterCompile"))
			})
		})

		Context("app has no Procfile", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "no_procfile", "src", "no_procfile"))
			})

			It("", func() {
				PushAppAndConfirm(app)
				Expect(app.GetBody("/")).To(ContainSubstring("go, world"))
				Expect(app.Stdout.String()).To(MatchRegexp(`Installing go [\d\.]+`))
				Expect(app.Stdout.String()).To(MatchRegexp(`Copy \[\/tmp\/`))
			})

			AssertNoInternetTraffic("no_procfile/src/no_procfile")
		})

		Context("expects a non-packaged version of go", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "go99"))
				resource_url = "https://storage.googleapis.com/golang/go99.99.99.linux-amd64.tar.gz"
			})

			It("displays useful understandable errors", func() {
				Expect(app.Push()).ToNot(BeNil())

				Expect(app.Stdout.String()).To(MatchRegexp("failed"))
				Expect(app.Stdout.String()).To(MatchRegexp("Unable to determine Go version to install: no match found for 99.99.99"))

				Expect(app.Stdout.String()).ToNot(MatchRegexp("Installing go99.99.99"))
				Expect(app.Stdout.String()).ToNot(MatchRegexp("Uploading droplet"))
			})
		})

		Context("heroku example", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "heroku_example"))
			})

			It("", func() {
				PushAppAndConfirm(app)
				Expect(app.GetBody("/")).To(ContainSubstring("hello, heroku"))
			})
			AssertNoInternetTraffic("heroku_example")
		})

		Context("a go app using ldflags with version 1.6~", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "go16_ldflags", "src", "go_app"))
			})

			It("", func() {
				PushAppAndConfirm(app)

				Expect(app.GetBody("/")).To(ContainSubstring("flag_linked"))
				Expect(app.Stdout.String()).To(ContainSubstring("main.linker_flag=flag_linked"))
			})

			AssertNoInternetTraffic("go16_ldflags/src/go_app")
		})

		Context("app uses glide and has vendored dependencies", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "glide_and_vendoring", "src", "glide_and_vendoring"))
			})

			It("", func() {
				PushAppAndConfirm(app)
				Expect(app.GetBody("/")).To(ContainSubstring("hello, world"))
				Expect(app.Stdout.String()).To(ContainSubstring("Hello from foo!"))
				Expect(app.Stdout.String()).To(ContainSubstring("Note: skipping (glide install) due to non-empty vendor directory."))
			})

			AssertNoInternetTraffic("glide_and_vendoring/src/glide_and_vendoring")
		})

		Context("go 1.7 app with GO_SETUP_GOPATH_IN_IMAGE", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "gopath_in_container", "src", "go_app"))
				app.SetEnv("GO_SETUP_GOPATH_IN_IMAGE", "true")
			})

			It("", func() {
				PushAppAndConfirm(app)
				Expect(app.GetBody("/")).To(ContainSubstring("GOPATH: /home/vcap/app"))
			})
		})

		Context("go 1.7 app with GO_INSTALL_TOOLS_IN_IMAGE", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "toolchain_in_container", "src", "go_app"))
				app.SetEnv("GO_INSTALL_TOOLS_IN_IMAGE", "true")
			})

			It("displays the go version", func() {
				PushAppAndConfirm(app)

				Expect(app.GetBody("/")).To(ContainSubstring("go version go1.7.6 linux/amd64"))
			})

			Context("running a task", func() {
				BeforeEach(func() {
					if !ApiHasTask() {
						Skip("Running against CF without run task support")
					}
				})

				It("can find the specifed go in the container", func() {
					PushAppAndConfirm(app)

					_, err := app.RunTask(`echo "RUNNING A TASK: $(go version)"`)
					Expect(err).ToNot(HaveOccurred())

					Eventually(func() string { return app.Stdout.String() }, 10*time.Second).Should(ContainSubstring("RUNNING A TASK: go version go1.7.6 linux/amd64"))
				})
			})

			Context("and GO_SETUP_GOPATH_IN_IMAGE", func() {
				BeforeEach(func() {
					app.SetEnv("GO_INSTALL_TOOLS_IN_IMAGE", "true")
					app.SetEnv("GO_SETUP_GOPATH_IN_IMAGE", "true")
				})

				It("displays the go version", func() {
					PushAppAndConfirm(app)

					Expect(app.GetBody("/")).To(ContainSubstring("go version go1.7.6 linux/amd64"))
					Expect(app.GetBody("/gopath")).To(ContainSubstring("GOPATH: /home/vcap/app"))
				})
			})
		})
	})
	Context("without cached buildpack dependencies", func() {
		BeforeEach(func() {
			if cutlass.Cached {
				Skip("but running cached tests")
			}
		})

		Context("app uses glide", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_glide", "src", "with_glide"))
			})

			It("", func() {
				PushAppAndConfirm(app)
				Expect(app.Stdout.String()).To(ContainSubstring("Hello from foo!"))
				Expect(app.GetBody("/")).To(ContainSubstring("hello, world"))
			})
			AssertUsesProxyDuringStagingIfPresent("with_glide/src/with_glide")
		})

		Context("app has dependencies", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_dependencies", "src", "with_dependencies"))
			})

			It("", func() {
				PushAppAndConfirm(app)
				Expect(app.Stdout.String()).To(MatchRegexp("Hello from foo!"))
				Expect(app.GetBody("/")).To(ContainSubstring("hello, world"))
			})

			AssertUsesProxyDuringStagingIfPresent("with_dependencies/src/with_dependencies")
		})

		Context("app has vendored dependencies", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "go17_vendor_experiment_flag"))
			})

			It("", func() {
				Expect(app.Push()).ToNot(BeNil())
				Expect(app.Stdout.String()).To(ContainSubstring("GO15VENDOREXPERIMENT is set, but is not supported by go1.7"))
			})
		})

		Context("app has no dependencies", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "go_app"))
			})

			It("", func() {
				PushAppAndConfirm(app)
				Expect(app.GetBody("/")).To(ContainSubstring("go, world"))
				Expect(app.Stdout.String()).To(MatchRegexp(`Installing go [\d\.]+`))
				Expect(app.Stdout.String()).To(MatchRegexp(`Download \[https:\/\/.*\]`))
			})
		})

		Context("expects a non-packaged version of go", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "go99"))
				resource_url = "https://storage.googleapis.com/golang/go99.99.99.linux-amd64.tar.gz"
			})

			It("displays useful understandable errors", func() {
				Expect(app.Push()).ToNot(BeNil())

				Expect(app.Stdout.String()).To(MatchRegexp("failed"))
				Expect(app.Stdout.String()).To(MatchRegexp("Unable to determine Go version to install: no match found for 99.99.99"))

				Expect(app.Stdout.String()).ToNot(MatchRegexp("Installing go99.99.99"))
				Expect(app.Stdout.String()).ToNot(MatchRegexp("Uploading droplet"))
			})
		})

		Context("heroku example", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "heroku_example"))
			})

			It("", func() {
				PushAppAndConfirm(app)
				Expect(app.GetBody("/")).To(ContainSubstring("hello, heroku"))
			})
			AssertUsesProxyDuringStagingIfPresent("heroku_example")
		})

		Context("a go app using ldflags with version 1.6~", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "go16_ldflags", "src", "go_app"))
			})

			It("", func() {
				PushAppAndConfirm(app)

				Expect(app.GetBody("/")).To(ContainSubstring("flag_linked"))
				Expect(app.Stdout.String()).To(ContainSubstring("main.linker_flag=flag_linked"))
			})
		})

		Context("app uses glide and has vendored dependencies", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "glide_and_vendoring", "src", "glide_and_vendoring"))
			})

			It("", func() {
				PushAppAndConfirm(app)
				Expect(app.GetBody("/")).To(ContainSubstring("hello, world"))
				Expect(app.Stdout.String()).To(ContainSubstring("Hello from foo!"))
				Expect(app.Stdout.String()).To(ContainSubstring("Note: skipping (glide install) due to non-empty vendor directory."))
			})

			AssertUsesProxyDuringStagingIfPresent("glide_and_vendoring/src/glide_and_vendoring")
		})

		Context("buildpack is not compiled eg. when source used from github", func() {
			var buildpack_language string

			BeforeEach(func() {
				buildpack_language = fmt.Sprintf("go-unpackaged-%s", cutlass.RandStringRunes(10))
				buildpack_file := fmt.Sprintf("/tmp/%s.zip", buildpack_language)

				cmd := exec.Command("zip", "-r", buildpack_file, "bin/", "src/", "scripts/", "manifest.yml", "VERSION")
				cmd.Dir = bpDir
				_, err := cmd.Output()
				Expect(err).ToNot(HaveOccurred())
				err = cutlass.CreateOrUpdateBuildpack(buildpack_language, buildpack_file)
				os.Remove(buildpack_file)
				Expect(err).ToNot(HaveOccurred())

				app = cutlass.New(filepath.Join(bpDir, "fixtures", "go_app"))
				app.Buildpack = fmt.Sprintf("%s_buildpack", buildpack_language)
			})

			AfterEach(func() {
				cutlass.DeleteBuildpack(buildpack_language)
			})

			It("runs apps", func() {
				Expect(app.Push()).To(Succeed())
				Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))
				buildpackVersion, err := ioutil.ReadFile(filepath.Join(bpDir, "VERSION"))
				Expect(err).NotTo(HaveOccurred())
				Expect(app.ConfirmBuildpack(string(buildpackVersion))).To(Succeed())

				Expect(app.Stdout.String()).To(ContainSubstring("Running go build supply"))
				Expect(app.Stdout.String()).To(ContainSubstring("Running go build finalize"))

				Expect(app.GetBody("/")).To(ContainSubstring("go, world"))
			})
		})

		Context("a .godir file is detected", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "deprecated_heroku", "src", "deprecated_heroku"))
			})

			It("fails with a deprecation message", func() {
				Expect(app.Push()).ToNot(BeNil())
				Expect(app.Stdout.String()).To(ContainSubstring("Deprecated, .godir file found! Please update to supported Godep or Glide dependency managers."))
				Expect(app.Stdout.String()).To(ContainSubstring("See https://github.com/tools/godep or https://github.com/Masterminds/glide for usage information."))
			})
		})

		Context("a go app with wildcard matcher", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "wildcard_go_version", "src", "go_app"))
			})

			It("fails with a deprecation message", func() {
				PushAppAndConfirm(app)
				Expect(app.GetBody("/")).To(ContainSubstring("go, world"))
				Expect(app.Stdout.String()).To(MatchRegexp(`Installing go 1\.6\.\d+`))
			})
		})

		Context("a go app with an invalid wildcard matcher", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "invalid_wildcard_version", "src", "go_app"))
			})

			It("fails with a helpful warning message", func() {
				Expect(app.Push()).ToNot(BeNil())
				Expect(app.Stdout.String()).To(MatchRegexp(`Unable to determine Go version to install: no match found for 1.3.x`))
				Expect(app.Stdout.String()).ToNot(MatchRegexp(`Installing go1.3`))
			})
		})

		Context("a go 1.6 app", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "go16", "src", "go_app"))
			})

			It("should be compiled with buildmode=pie", func() {
				PushAppAndConfirm(app)
				Expect(app.GetBody("/")).To(MatchRegexp(`foo: .*`))
				app_body, _ := app.GetBody("/")
				Restart(app)
				Expect(app.GetBody("/")).To(MatchRegexp(`foo: .*`))
				Expect(app.GetBody("/")).ToNot(Equal(app_body))
			})
		})
	})
})
