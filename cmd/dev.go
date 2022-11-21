package cmd

import (
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/spf13/cobra"
)

func NewDevCommand() *cobra.Command {
	devCommand := &cobra.Command{
		Use:   "dev",
		Short: "",
		RunE:  runDev,
	}
	return devCommand
}

func runDev(cmd *cobra.Command, args []string) error {

	data := make(map[string]string)

	data["testing"] = "123"

	err := k8s.CreateMapStringSecret("vault", "vault-tls", data)
	if err != nil {
		return err
	}

	//err := k8s.IngressCreate("vault", "vault", 8200)
	//if err != nil {
	//	return err
	//}
	//err := k8s.IngressDelete("vault", "vault")
	//if err != nil {
	//	return err
	//}
	//err := k8s.IngressAddRule("default", "k3d-ingress-rules", "vault", 8200)
	//if err != nil {
	//	return err
	//}

	// priv, err := rsa.GenerateKey(rand.Reader, *rsaBits)
	//priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//template := x509.Certificate{
	//	SerialNumber: big.NewInt(1),
	//	Subject: pkix.Name{
	//		Organization: []string{"Kubefirst"},
	//	},
	//	IsCA:      true,
	//	NotBefore: time.Now(),
	//	NotAfter:  time.Now().Add(time.Hour * 24 * 180),
	//
	//	KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	//	ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	//	BasicConstraintsValid: true,
	//}

	/*
	   hosts := strings.Split(*host, ",")
	   for _, h := range hosts {
	   	if ip := net.ParseIP(h); ip != nil {
	   		template.IPAddresses = append(template.IPAddresses, ip)
	   	} else {
	   		template.DNSNames = append(template.DNSNames, h)
	   	}
	   }

	   if *isCA {
	   	template.IsCA = true
	   	template.KeyUsage |= x509.KeyUsageCertSign
	   }
	*/

	//derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	//if err != nil {
	//	log.Fatalf("Failed to create certificate: %s", err)
	//}
	//out := &bytes.Buffer{}
	//pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	//fmt.Println(out.String())
	//out.Reset()
	//pem.Encode(out, pemBlockForKey(priv))
	//fmt.Println(out.String())
	//
	return nil
}

//func publicKey(priv any) any {
//	switch k := priv.(type) {
//	case *rsa.PrivateKey:
//		return &k.PublicKey
//	case *ecdsa.PrivateKey:
//		return &k.PublicKey
//	default:
//		return nil
//	}
//}
//
//func pemBlockForKey(priv interface{}) *pem.Block {
//	switch k := priv.(type) {
//	case *rsa.PrivateKey:
//		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
//	case *ecdsa.PrivateKey:
//		b, err := x509.MarshalECPrivateKey(k)
//		if err != nil {
//			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
//			os.Exit(2)
//		}
//		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
//	default:
//		return nil
//	}
//}
