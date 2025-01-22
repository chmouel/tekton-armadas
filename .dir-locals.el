
((nil . ((eval . (add-to-list 'dape-configs
                              `(orchestrator-watcher modes (go-mode go-ts-mode) ensure dape-ensure-command fn dape-config-autoport
                                               command "dlv" command-args ("dap" "--listen" "127.0.0.1::autoport" "--log=true") command-cwd dape-cwd-fn port :autoport
                                               :type "debug"
                                               :request "launch" 
                                               :args
                                               ["--disable-ha"]
                                               :program "./cmd/orchestrator-reconciler/"))))))
