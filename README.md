# glacier

glacier is a app framework for rapid service development

Usage:

    go get github.com/mylxsw/glacier

Demo:

	app := application.Create("v1.0")

	g := app.Glacier()
	g.WithHttpServer(":19945")

	g.WebAppExceptionHandler(func(ctx web.Context, err interface{}) web.Response {
		log.Errorf("stack: %s", debug.Stack())
		return nil
	})

	g.Provider(job.ServiceProvider{})
	g.Provider(api.ServiceProvider{})

	g.Service(&service.DemoService{})
	g.Service(&service.Demo2Service{})

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}