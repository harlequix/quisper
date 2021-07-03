from invoke import task, run
import pathlib
from glob import glob
import shutil
import logging as log
import regex as re
from eval.utils import parse_folder_name
from pprint import pprint


log.basicConfig(format='%(levelname)s:%(message)s', level=log.INFO)

DATA = pathlib.Path("./data/")

RUNBASE = pathlib.Path("./runs/")
LOGS = "logs"
STATS = "stats"
RESULTS = "results"


@task
def cleanup(ctx):
    cmd = f"grep -r  'Test run stuck for too long, abort' ./runs"
    run(cmd)


@task
def build(ctx):
    print("Building!")


@task
def init(ctx, name):
    if isinstance(name, str):
        name = RUNBASE/name
    touch(name)
    traces = gather(name)
    for trace in traces:
        shutil.move(trace, name/LOGS)


@task
def update(ctx, paths="all"):
    runs = get_paths(paths)
    keys = ["rx", "tx", "server", "payload", "config"]
    results = [x.stem for x in DATA.glob("*")]
    print(results)
    parsed = []
    for x in runs:
        entry = parse_folder_name(x.name)
        entry["base"] = x
        parsed.append(entry)
    parsed = combine(parsed, keys)
    for result in results:
        for group in parsed:
            if len(group) > 1:
                df = parsed[group][0]
            else:
                log.Error("not implemented")
                df = parsed[group][0]
            source = df["base"]/RESULTS/f"{result}.csv"
            target = DATA/result/f"data/{group}.csv"
            if not source.is_file():
                log.error(f"{source} does not exist")
            else:
                target.unlink(missing_ok=True)
                source.link_to(target)


@task()
def stats(ctx, paths, rebuild=False):
    runs = get_paths(paths)
    # print(runs)
    for runP in runs:
        init(ctx, runP)
        pairs = pair_up(runP/LOGS)
        for pair in pairs:
            output = pathlib.Path(f"{runP/STATS/pair}.csv")
            data = pairs[pair]
            if is_valid(data) and (rebuild or not output.is_file()):
                try:
                    cmd = f"python eval/stats.py --traces {data[0]} {data[1]} --out {output}"
                    run(cmd)
                except IndexError:
                    print(data)


@task
def throughput(ctx, name):
    runs = get_paths(name)
    for r in runs:
        touch(r)
        stats = (r/STATS).glob("*.csv")
        for stat in stats:
            cmd = f"python eval/throughput.py  {stat} {r/RESULTS/(f'throughput-{stat.name}')}"
            try:
                run(cmd)
            except Exception as e:
                print(stat)
                print(e)


@task
def walltime(ctx, name, rebuild=False):
    runs = get_paths(name)
    for r in runs:
        touch(r)
        stats = (r/STATS).glob("*.csv")
        stats = [str(x) for x in stats]
        output = r/RESULTS/"walltime.csv"
        if rebuild or not output.is_file():
            statsStr = " ".join(stats)
            cmd = f"python eval/walltime.py {statsStr} {output}"
            try:
                run(cmd)
            except Exception as e:
                log.warn(f"run:{r}, cmd:{cmd}, error:{e}")

@task
def prep_accuracy(ctx, name, rebuild=False):
    experiments = get_paths(name)
    for exp in experiments:
        stats = (exp/STATS).glob("*.csv")
        acc_result = (exp/RESULTS/"accuracy")
        acc_result.mkdir(parents=True, exist_ok=True)
        for stat in stats:
            output = acc_result/f"accuracy-{stat.name}"
            cmd = f"python eval/accuracy.py {stat} {output}"
            run(cmd)



def get_paths(name):
    if name == "all":
        name = ""
    name = name + "*"
    paths = [x for x in RUNBASE.glob(name)]
    return paths


def pair_up(path):
    pairs = {}
    files = path.glob("*")
    for file in files:
        log.debug(file)
        name = file.name
        log.debug(name)
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
    stats = base / RESULTS
    stats.mkdir(exist_ok=True)


def is_valid(paths):
    # valid = True
    expr = re.compile("Test run stuck for too long, abort")
    for path in paths:
        with open(path) as checkfile:
            for line in checkfile:
                match = expr.search(line)
                if match:
                    log.warn(f"Run {checkfile.name} is broken")
                    return False
    return True
    # cmd = f"grep -r  'Test run stuck for too long, abort'"
    # ps = [str(x.resolve()) for x in paths]
    # print(ps)
    # cmd = [cmd]
    # cmd += ps
    # print(cmd)
    # foobar = " ".join(cmd)
    # foobar += " | cat"
    # result = run(foobar, hide="out")
    # if result.stdout:
    #     return False
    # return True


def combine(data, keywords):
    out = {}
    for entry in data:
        key = "-".join([entry[x] for x in keywords])
        if key not in out.keys():
            out[key] = []
        out[key].append(entry)
    return out
