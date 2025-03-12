import asyncio
import string
import time
from itertools import product
from typing import List

async def generate_passwords(charset: List[str], max_length: int):
    for length in range(1, max_length + 1):
        for pwd_tuple in product(charset, repeat=length):
            yield ''.join(pwd_tuple)

async def test_passwords(charset: List[str], max_length: int):
    start_time = time.time()
    total_passwords_tested = 0

    async for password in generate_passwords(charset, max_length):
        total_passwords_tested += 1
        await asyncio.sleep(0)

        if total_passwords_tested % 10000 == 0:
            elapsed_time = time.time() - start_time
            passwords_per_second = total_passwords_tested / elapsed_time
            print(f"Tested {total_passwords_tested} passwords at {passwords_per_second:.2f} passwords per second")
        del password

def get_charset(option: str) -> List[str]:
    options = {
        '1': string.digits,
        '2': string.ascii_letters,
        '3': string.ascii_letters + string.digits,
        '4': string.ascii_letters + string.digits + string.punctuation
    }
    return list(options.get(option, ""))

async def main():
    print("Choose character set:")
    print("1. Numbers only")
    print("2. Letters only")
    print("3. Letters and numbers")
    print("4. Letters, numbers, and special characters")

    option = input("Enter your choice (1-4): ")
    charset = get_charset(option)
    max_length = int(input("Enter maximum password length: "))

    await test_passwords(charset, max_length)

if __name__ == '__main__':
    asyncio.run(main())
