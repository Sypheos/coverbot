package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v41/github"
	"github.com/upfluence/cfg/x/cli"
	"github.com/upfluence/errors"
	"golang.org/x/oauth2"
)

type repository struct {
	owner string
	name  string
}

func (r *repository) Parse(s string) error {
	ss := strings.Split(s, "/")

	if len(ss) != 2 {
		return errors.New("expect repository format owner/repo_name")
	}

	r.owner = ss[0]
	r.name = ss[1]
	return nil
}

type config struct {
	File  string     `flag:"cov,coverage-file"`
	Repo  repository `flag:"repo"`
	Issue int        `flag:"issue"`
	Diff  string     `flag:"diff,diff-from"`
}

func main() {
	cli.NewApp(
		cli.WithName("coverbot"),
		cli.WithCommand(
			cli.StaticCommand{
				Help:     cli.StaticString("post coverage"),
				Synopsis: cli.SynopsisWriter(&config{}),
				Execute: func(ctx context.Context, cctx cli.CommandContext) error {
					var cfg = config{File: "cover.out"}

					if err := cctx.Configurator.Populate(ctx, &cfg); err != nil {
						return err
					}

					if cfg.Repo.owner == "" || cfg.Repo.name == "" || cfg.Issue == 0 {
						return errors.New("missing repo or issue info")
					}

					coverage, err := funcOutput(cfg.File)

					if err != nil {
						return err
					}

					comment := fmt.Sprintf("total coverage: %.2f", coverage)

					if cfg.Diff != "" {
						diffCoverage, err := funcOutput(cfg.Diff)

						if err != nil {
							return err
						}

						d := coverage - diffCoverage

						if d >= 0 {
							comment = fmt.Sprintf("Coverage increased from %.2f to %.2f (+%.3f)", diffCoverage, coverage, d)
						} else {
							comment = fmt.Sprintf("Coverage decreased from %.2f to %.2f (-%.3f)", diffCoverage, coverage, d)
						}
					}

					ts := oauth2.StaticTokenSource(
						&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
					)
					tc := oauth2.NewClient(ctx, ts)

					ghcl := github.NewClient(tc).Issues

					_, _, err = ghcl.CreateComment(
						ctx,
						cfg.Repo.owner,
						cfg.Repo.name,
						cfg.Issue,
						&github.IssueComment{
							Body: &comment,
						},
					)

					if err != nil {
						return err
					}

					return nil
				},
			},
		),
	).Run(context.Background())
}
