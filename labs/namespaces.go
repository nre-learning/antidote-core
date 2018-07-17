// var kubeconfig *string
// if home := homeDir(); home != "" {
// 	kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
// } else {
// 	kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
// }
// flag.Parse()

// // use the current context in kubeconfig
// config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
// if err != nil {
// 	panic(err.Error())
// }

// // create the clientset
// clientset, err := kubernetes.NewForConfig(config)
// if err != nil {
// 	panic(err.Error())
// }

// existingNamespaces, err := clientset.Core().Namespaces().List(metav1.ListOptions{})
// if err != nil {
// 	panic(err.Error())
// }
// fmt.Printf("EXISTING NAMESPACES: %s", existingNamespaces)

// nsSpec := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "antidote-lesson1-abcdef"}}
// _, err = clientset.Core().Namespaces().Create(nsSpec)
package labs
