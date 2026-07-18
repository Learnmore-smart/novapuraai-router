import { toast } from 'sonner'

export function showSubmittedData(
  data: unknown,
  title: string = 'You submitted the following values:'
) {
  toast.message(title, {
    description: (
      // w-[340px]
      <pre className='bg-muted text-foreground mt-2 w-full overflow-x-auto rounded-md p-4'>
        <code>{JSON.stringify(data, null, 2)}</code>
      </pre>
    ),
  })
}
