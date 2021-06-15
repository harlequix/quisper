from invoke import task, run
import pathlib
from glob import glob
import shutil


RUNBASE = pathlib.Path("./runs/")
LOGS = "logs"
STATS = "stats"
RESULTS = "results"


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
    init(ctx, name)
    pairs = pair_up(RUNBASE/name/LOGS)
    print(pairs)
    for pair in pairs:
        data = pairs[pair]
        cmd = f"python eval/stats.py --traces {data[0]} {data[1]} --out {RUNBASE/name/STATS/pair}.csv"
        print(cmd)
        run(cmd)


@task
def throughput(ctx, name):
    stats = (RUNBASE/name/STATS).glob("*.csv")
    for stat in stats:
        cmd = f"python eval/throughput.py  {stat} {RUNBASE/name/RESULTS/(f'throughput-{stat.name}')}"
        run(cmd)


def pair_up(path):
    pairs = {}
    files = path.glob("*")
    for file in files:
        print(file)
        name = file.name
        print(name)
        key = name.rsplit("-", 1)[0]
        if key not in pairs:
            pairs[key] = []
        pairs[key].append(file)
    return pairs


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
