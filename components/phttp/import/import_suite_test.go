package phttp

import (
	"net"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	. "github.com/yandex/pandora/components/phttp"
	"github.com/yandex/pandora/lib/ginkgoutil"
)

func TestImport(t *testing.T) {
	ginkgoutil.RunSuite(t, "phttp Import Suite")
}

var _ = Describe("import", func() {
	It("not panics", func() {
		Expect(func() {
			Import(afero.NewOsFs())
		}).NotTo(Panic())
	})
})

var _ = Describe("preResolveTargetAddr", func() {
	It("host target", func() {
		conf := &ClientConfig{}
		conf.Dialer.DNSCache = true

		listener, err := net.ListenTCP("tcp4", nil)
		if listener != nil {
			defer listener.Close()
		}
		Expect(err).NotTo(HaveOccurred())

		port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
		target := "localhost:" + port
		expectedResolved := "127.0.0.1:" + port

		err = preResolveTargetAddr(conf, &target)
		Expect(err).NotTo(HaveOccurred())
		Expect(conf.Dialer.DNSCache).To(BeFalse())

		Expect(target).To(Equal(expectedResolved))
	})

	It("ip target", func() {
		conf := &ClientConfig{}
		conf.Dialer.DNSCache = true

		const addr = "127.0.0.1:80"
		target := addr
		err := preResolveTargetAddr(conf, &target)
		Expect(err).NotTo(HaveOccurred())
		Expect(conf.Dialer.DNSCache).To(BeFalse())
		Expect(target).To(Equal(addr))
	})

	It("failed", func() {
		conf := &ClientConfig{}
		conf.Dialer.DNSCache = true

		const addr = "localhost:54321"
		target := addr
		err := preResolveTargetAddr(conf, &target)
		Expect(err).To(HaveOccurred())
		Expect(conf.Dialer.DNSCache).To(BeTrue())
		Expect(target).To(Equal(addr))
	})

})
