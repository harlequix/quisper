import argparse
import json


def read_goson(jsonPath):
    out = []
    with open(jsonPath) as jsonFile:
        for line in jsonFile:
            parsed = json.loads(line)
            out.append(parsed)
    return out


def match(entry, filter):
    val = [entry[x] == filter[x] for x in filter]
    return all(val)


def filterData(dataset, filter):
    out = [x for x in dataset if match(x, filter)]
    return out


def check_accuracy(data1, data2):
    filter = {
        "msg": "Finished Request"
    }
    data1 = filterData(data1, filter)
    data2 = filterData(data2, filter)
    unified = {}
    for i in data1:
        key = i["cid"]
        if key not in unified:
            unified[key] = []
        unified[key].append(i["result"])
    for i in data2:
        key = i["cid"]
        if key not in unified:
            unified[key] = []
        unified[key].append(i["result"])
    correct_cnt = 0
    total = 0
    for key in sorted(unified):
        if len(unified[key]) == 2:
            total += 1
            if unified[key][0] != unified[key][1]:
                correct_cnt += 1
            else:
                print(key, unified[key])
    return correct_cnt/total


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("rxFile")
    parser.add_argument("txFile")
    args = parser.parse_args()
    rxData = read_goson(args.rxFile)
    txData = read_goson(args.txFile)
    accuraccy = check_accuracy(rxData, txData)
    print(accuraccy)


if __name__ == '__main__':
    main()
