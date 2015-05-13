package backup

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("backup", func() {

	Describe(".cleanup", func() {
		var tempRdbFile *os.File

		BeforeEach(func() {
			var err error
			tempRdbFile, err = ioutil.TempFile("", "temp")
			Expect(err).ToNot(HaveOccurred())
		})

		Context("RDB file does not exist", func() {
			It("renames the temp RDB", func() {
				dir, err := ioutil.TempDir("", "")
				Expect(err).ToNot(HaveOccurred())

				rdbPath := filepath.Join(dir, "dump.rdb")

				Expect(fileExists(rdbPath)).To(BeFalse())

				cleanup(tempRdbFile.Name(), rdbPath)

				Expect(fileExists(tempRdbFile.Name())).To(BeFalse())
				Expect(fileExists(rdbPath)).To(BeTrue())
			})
		})

		Context("RDB file exists", func() {
			var rdbPath string

			BeforeEach(func() {
				rdbFile, err := ioutil.TempFile("", "rdb")
				Expect(err).ToNot(HaveOccurred())
				_, err = rdbFile.WriteString("something")
				Expect(err).ToNot(HaveOccurred())
				rdbPath = rdbFile.Name()
				rdbFile.Close()
			})

			It("does not touch the existing RDB", func() {
				cleanup(tempRdbFile.Name(), rdbPath)

				Expect(fileExists(rdbPath)).To(BeTrue())
				fileContents, err := ioutil.ReadFile(rdbPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(fileContents)).To(Equal("something"))
			})

			It("deletes the temp RDB", func() {
				cleanup(tempRdbFile.Name(), rdbPath)
				Expect(fileExists(tempRdbFile.Name())).To(BeFalse())
			})
		})
	})
})
