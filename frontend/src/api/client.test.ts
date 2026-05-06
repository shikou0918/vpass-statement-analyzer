import { afterEach, describe, expect, it, vi } from 'vitest'
import { createCategory, listCategories } from './client'

describe('api client', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('requests categories and parses the list response', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ items: [{ id: 1, name: '食費', color: '#22c55e' }] }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const result = await listCategories()

    expect(fetchMock).toHaveBeenCalledWith('http://localhost:8080/categories', {
      headers: { 'Content-Type': 'application/json' },
    })
    expect(result.items).toEqual([{ id: 1, name: '食費', color: '#22c55e' }])
  })

  it('normalizes backend error responses', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ code: 'BAD_REQUEST', message: 'カテゴリ名は必須です', details: { field: 'name' } }), {
          status: 400,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    )

    await expect(createCategory({ name: '', color: 'blue' })).rejects.toMatchObject({
      apiError: {
        code: 'BAD_REQUEST',
        message: 'カテゴリ名は必須です',
        details: { field: 'name' },
      },
    })
  })
})
