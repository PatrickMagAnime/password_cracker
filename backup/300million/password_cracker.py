import string
import time
import math
from multiprocessing import Pool, cpu_count, Manager
from tqdm import tqdm

def index_to_password(idx, charset, length):
    password = []
    cs_len = len(charset)
    for _ in range(length):
        idx, rem = divmod(idx, cs_len)
        password.append(charset[rem])
    return ''.join(reversed(password))

def check_range(args):
    start, end, charset, length, target, found_flag = args
    if found_flag.value:
        return 0, 0
    
    cs_len = len(charset)
    batch_size = 5000
    tested = 0
    
    for idx in range(start, end + 1, batch_size):
        if found_flag.value:
            return tested, 0
        
        batch_end = min(idx + batch_size - 1, end)
        tested += batch_end - idx + 1
        
        for i in range(idx, batch_end + 1):
            if index_to_password(i, charset, length) == target:
                found_flag.value = 1
                return tested, 1
    
    return tested, 0

def main():
    target = input("Zielpasswort: ")
    
    print("\nZeichensatzauswahl:")
    print("1. Nur Zahlen\n2. Nur Buchstaben\n3. Buchstaben + Zahlen\n4. Alle Zeichen")
    charset = {
        '1': string.digits,
        '2': string.ascii_letters,
        '3': string.ascii_letters + string.digits,
        '4': string.ascii_letters + string.digits + string.punctuation
    }.get(input("Wahl (1-4): "), "")
    
    if not charset:
        print("Ungültige Eingabe!")
        return
    
    max_length = int(input("Maximale Länge: "))
    charset = tuple(charset)
    cs_len = len(charset)
    total_combinations = sum(cs_len**l for l in range(1, max_length+1))
    
    start_time = time.time()
    manager = Manager()
    found_flag = manager.Value('i', 0)
    total_tested = 0
    
    with tqdm(total=total_combinations, desc="Gesamtfortschritt", unit="pwd",
             bar_format="{l_bar}{bar}| {n_fmt}/{total_fmt} [{elapsed}<{remaining}, {rate_fmt}{postfix}]") as pbar:
        with Pool(processes=cpu_count()) as pool:
            for length in range(1, max_length + 1):
                if found_flag.value:
                    break
                
                total = cs_len ** length
                chunk_size = max(1000, total // (cpu_count() * 10))
                chunks = []
                
                for start in range(0, total, chunk_size):
                    end = min(start + chunk_size - 1, total - 1)
                    chunks.append((
                        start, 
                        end, 
                        charset, 
                        length, 
                        target, 
                        found_flag
                    ))
                
                results = pool.imap_unordered(check_range, chunks, chunksize=10)
                
                for tested, found in results:
                    total_tested += tested
                    pbar.update(tested)
                    if found:
                        found_flag.value = 1
                        pool.terminate()
                        break

    print("\n" + "="*50)
    print(f"Passwort {'gefunden' if found_flag.value else 'nicht gefunden'}")
    
    if found_flag.value:
        elapsed_time = time.time() - start_time
        print(f"Gesamtzeit: {elapsed_time:.2f}s")
        print(f"Getestete Passwörter: {total_tested}")
        print(f"Durchschnittliche Geschwindigkeit: {total_tested / elapsed_time:,.0f} Passwörter/Sekunde")

if __name__ == "__main__":
    main()