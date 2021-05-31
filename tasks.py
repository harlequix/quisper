from invoke import task, run
import pathlib
from glob import glob
import shutil


RUNBASE = pathlib.Path("./runs/")
LOGS = "logs"
STATS = "stats"


@task
def build(ctx):
    print("Building!")


@task
def init(ctx, name):
    touch(RUNBASE/name)
    traces = gather(name)
    for trace in traces:
        shutil.move(trace, RUNBASE/name/LOGS)


@task
def stats(ctx, name):
    pairs = pair_up(RUNBASE/name/LOGS)
    for pair in pairs:
        splits = pair[0].name.split("-")
        runname = "-".join([splits[1], splits[2]])
        cmd = f"python eval/stats.py --traces {pair[0]} {pair[1]} --out {RUNBASE/name/STATS/runname}.csv"
        print(cmd)
        run(cmd)


def pair_up(path):
    pairs = {}
    files = path.glob("*")
    for file in files:
        print(file)
        name = file.name
        key = name[1:]
        if key not in pairs:
            pairs[key] = []
        pairs[key].append(file)
    return pairs.values()


def gather(name):
    expr = f"./[r,t]x-{name}*"
    files = glob(expr)
    return files


def touch(base):
    base.mkdir(exist_ok=True)
    logs = base / LOGS
    logs.mkdir(exist_ok=True)
    stats = base / STATS
    stats.mkdir(exist_ok=True)
