from invoke import task, run
import pathlib
from glob import glob
import shutil
import logging as log
import regex as re
from eval.utils import parse_folder_name
from pprint import pprint
from eval.validate_logs import validate_logfile
import argparse
import multiprocessing

log.basicConfig(format='%(levelname)s:%(message)s', level=log.INFO)

DATA = pathlib.Path("./data")

parser = argparse.ArgumentParser()
parser.add_argument("--runbase", default="./runs/")
args, _ = parser.parse_known_args()


RUNBASE = pathlib.Path(args.runbase)
LOGS = "logs"
STATS = "stats"
RESULTS = "results"


@task
def metrics(ctx, name, runbase=RUNBASE):
    stats(ctx, name, runbase=runbase)
    throughput(ctx, name, runbase=runbase)
    walltime(ctx, name, runbase=runbase)
    overhead(ctx, name, runbase=runbase)
    accuracy(ctx, name, runbase=runbase)


@task
def cleanup(ctx, paths="all"):
    runs = get_paths(paths)
    for run in runs:
        log.disable("WARN")
        logs = (run/LOGS).glob("*.log")
        faulty = [x for x in logs if not validate_logfile(x)]
        if faulty:
            print(run)
            for f in faulty:
                print(f"\t{f} is faulty")


@task
def avgtput(ctx, paths, rebuild=False, runbase=RUNBASE):
    runs = get_paths(paths)
    for r in runs:
        cmd = f"python eval/avg_tput.py {r}"
        print(cmd)
        run(cmd)

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
def stats(ctx, paths, rebuild=False, runbase=RUNBASE):
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
def throughput(ctx, name, runbase=RUNBASE):
    runs = get_paths(name)
    for r in runs:
        touch(r)
        rxlogs = (r/LOGS).glob("*rx.log")
        txlogs = (r/LOGS).glob("*tx.log")
        rxlogs = sorted([x for x in rxlogs])
        txlogs = sorted([x for x in txlogs])
        logs_filtered = [x for x in zip(rxlogs, txlogs) if is_valid(x)]
        rxinputs = " ".join([str(x[0]) for x in logs_filtered])
        txinputs = " ".join([str(x[1]) for x in logs_filtered])
        rxcmd = f"python eval/throughput-rx.py --inputs {rxinputs} --output {r/RESULTS/'throughput-rx.csv'}"
        txcmd = f"python eval/throughput-tx.py --inputs {txinputs} --output {r/RESULTS/'throughput-tx.csv'}"

        for cmd in [txcmd, rxcmd]:
            try:
                run(cmd)
            except Exception as e:
                print(cmd)
                print(e)


@task
def overhead(ctx, name, runbase=RUNBASE):
    runs = get_paths(name)
    for r in runs:
        cmd = f"python eval/overhead.py {r}"
        try:
            run(cmd)
        except Exception as e:
            log.warn(f"run:{r}, cmd:{cmd}, error:{e}")


@task
def walltime(ctx, name, runbase=RUNBASE, rebuild=False):
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
def accuracy(ctx, name, runbase=RUNBASE, rebuild=False):
    experiments = get_paths(name)
    for exp in experiments:
        stats = (exp/STATS).glob("*.csv")
        acc_result = (exp/RESULTS/"accuracy.csv")
        # acc_result.mkdir(parents=True, exist_ok=True)
        paths = " ".join([str(x) for x in stats])
        if paths != " ":
            cmd = f"python eval/accuracy.py --input {paths} --output {acc_result}"
            try:
                run(cmd)
            except Exception as e:
                log.warn(f"run:{exp}, cmd:{cmd}, error:{e}")


def get_paths(name):
    if name == "all":
        name = ""
    name = name + "*"
    paths = [x for x in RUNBASE.glob(name)]
    return paths


def pair_up(path):
    pairs = {}
    files = path.glob("*.log")
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
