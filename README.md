# eugène, version your system packages (and more!) in generations

eugène is a program that enables you to

- version your installations in generations
- switch back and forth between these generations

It achieves so by defining handlers for the entries you will define.

Let's say you want to manage apt packages on your system.
First, create the following configuration in `~/.eugene/eugene.yml` :

```yml
handlers:
  - name: apt_pkgs
    add: sudo apt install %s
    remove: sudo apt purge --autoremove %s
    # you can also specify the following fields
    sync: sudo apt update
    upgrade: sudo apt full-upgrade
    multiple: true # replace %s with several entries at a time
```

Then, in `~/.eugene`, create any file matching `apt_pkgs*` and put your packages, one per line.

You can now run `eugene build "an optionnal comment"`. Congratulations! You've just built the first generation.
Switch to your newly build generations by running `eugene switch latest`, your new packages will automatically install.

Every command of every handler you define in eugène is run as `sh -c "$command"`.
Therefore, you can use all your environment variables.

eugène also support hooks.
You can add `run_before_switch` and `run_after_switch` to any handler.

Give eugène a go and take a look at all the options by simply running `eugene`.
The help message will explain all you need to know and a sample configuration file will be (eu)generated.

### Not limited to system packages

As eugène can run any command on your entries, you can adapt it to all your needs.
eugène can, for example, be used to configure `gsettings` values for your Gnome configuration.

### Why eugène and not ansible?

Ansible is a great tool to ensure your host is configured the way you declared it.
Furthermore, ansible is idempotent.
But what ansible lacks is the ability to undo what your plays did.

eugène brings this ability and can ensure idempotency as well depending on your handlers.

I would still recommend to use ansible for server use and production environment.
For your day-to-day use on desktop/laptop, where configuration lives and evolves, I thing eugène fits better.

### Why eugène and not a script?

Scripts are great, but unless you're a very careful person, your script is probably a one-shot install.
Here again, eugène suits the life of a system, where things change and need to be undone.

## Installation

eugène is made in go

```sh
git clone https://github.com/yoannlr/eugene
cd eugene
go build
sudo cp eugene /usr/local/bin/
```

## Integration

You can very likely run eugene with [chezmoi](https://chezmoi.io). "Official" integration instructions coming soon!
