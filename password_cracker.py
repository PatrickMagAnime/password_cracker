import string
import time
import itertools
from concurrent.futures import ThreadPoolExecutor, as_completed
import tqdm

def check_password(candidate: str, target: str) -> bool:
    return candidate == target

def generate_passwords(charset, max_length):
    for length in range(1, max_length + 1):
        for pwd_tuple in itertools.product(charset, repeat=length):
            yield ''.join(pwd_tuple)

def chunk_generator(iterable, chunk_size):
    it = iter(iterable)
    while True:
        chunk = list(itertools.islice(it, chunk_size))
        if not chunk:
            break
        yield chunk

def check_chunk(candidates, target):
    for candidate in candidates:
        if check_password(candidate, target):
            return candidate
    return None

def main():
    target = input("Zielpasswort eingeben: ")
    
    print("Wähle Zeichensatz:")
    print("1. Nur Zahlen")
    print("2. Nur Buchstaben")
    print("3. Buchstaben und Zahlen")
    print("4. Buchstaben, Zahlen und Sonderzeichen")
    option = input("Deine Wahl (1-4): ")
    charset_options = {
        '1': string.digits,
        '2': string.ascii_letters,
        '3': string.ascii_letters + string.digits,
        '4': string.ascii_letters + string.digits + string.punctuation,
    }
    charset = charset_options.get(option, "")
    if not charset:
        print("Ungültige Wahl!")
        return
    charset = list(charset)
    max_length = int(input("Maximale Passwortlänge: "))

    total_combinations = sum(len(charset) ** i for i in range(1, max_length + 1))
    
    start_time = time.time()
    found_password = None
    tested_count = 0
    chunk_size = 1000 

    password_gen = generate_passwords(charset, max_length)
    
    with ThreadPoolExecutor(max_workers=8) as executor:
        with tqdm.tqdm(total=total_combinations, desc="Teste Passwörter...", unit="pwd") as pbar:
            futures = {executor.submit(check_chunk, chunk, target): len(chunk) for chunk in chunk_generator(password_gen, chunk_size)}
            
            for future in as_completed(futures):
                result = future.result()
                tested_count += futures[future]
                pbar.update(futures[future])
                elapsed = time.time() - start_time
                pps = tested_count / elapsed if elapsed > 0 else 0
                eta = (total_combinations - tested_count) / pps if pps > 0 else float('inf')
                pbar.set_postfix({"pps": f"{pps:.2f}", "ETA": f"{eta:.2f}s"})
                if result is not None:
                    found_password = result
                    break

    elapsed_time = time.time() - start_time
    print("\n--- Zusammenfassung ---")
    print(f"Getestete Passwörter: {tested_count}")
    print(f"Gesamtzeit: {elapsed_time:.2f} Sekunden")
    if found_password:
        print(f"Gefundenes Passwort: {found_password}")
    else:
        print("Kein Passwort gefunden.")
    
if __name__ == '__main__':
    main()
