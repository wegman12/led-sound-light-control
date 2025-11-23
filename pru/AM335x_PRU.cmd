/* AM335x PRU Linker Command File */

-cr
-stack 0x100
-heap 0x100

MEMORY
{
    PAGE 0:
      PRUIMEM : org = 0x00000000 len = 0x00002000  /* 8kB PRU Instruction RAM */

    PAGE 1:
      PRUDMEM : org = 0x00000000 len = 0x00002000  /* 8kB PRU Data RAM */

    PAGE 2:
      SHAREDMEM : org = 0x00010000 len = 0x00003000  /* 12kB Shared RAM */
}

SECTIONS
{
    .text          >  PRUIMEM, PAGE 0
    .stack         >  PRUDMEM, PAGE 1
    .bss           >  PRUDMEM, PAGE 1
    .cio           >  PRUDMEM, PAGE 1
    .data          >  PRUDMEM, PAGE 1
    .switch        >  PRUDMEM, PAGE 1
    .sysmem        >  PRUDMEM, PAGE 1
    .cinit         >  PRUDMEM, PAGE 1
    .rodata        >  PRUDMEM, PAGE 1
    .rofardata     >  PRUDMEM, PAGE 1
    .farbss        >  PRUDMEM, PAGE 1
    .fardata       >  PRUDMEM, PAGE 1
    .resource_table > PRUDMEM, PAGE 1
}
