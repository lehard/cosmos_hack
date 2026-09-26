/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */
import {
  useMutation,
  useQuery
} from '@tanstack/vue-query';
import type {
  DataTag,
  MutationFunction,
  QueryClient,
  QueryFunction,
  QueryKey,
  UseMutationOptions,
  UseMutationReturnType,
  UseQueryOptions,
  UseQueryReturnType
} from '@tanstack/vue-query';

import {
  toValue,
  unref
} from 'vue';
import type {
  MaybeRefOrGetter
} from 'vue';

import type {
  AccessPermissionListParams,
  DemoPersonaList,
  Desk,
  EntityChanged,
  IntegrityStatus,
  ItemItemLookupParams,
  ItemLookup,
  NotificationSummary,
  NotificationsSummaryReadParams,
  PermissionList,
  ProblemResponse,
  Session,
  SessionCreate
} from './model';


export type HTTPStatusCode1xx = 100 | 101 | 102 | 103;
export type HTTPStatusCode2xx = 200 | 201 | 202 | 203 | 204 | 205 | 206 | 207;
export type HTTPStatusCode3xx = 300 | 301 | 302 | 303 | 304 | 305 | 307 | 308;
export type HTTPStatusCode4xx = 400 | 401 | 402 | 403 | 404 | 405 | 406 | 407 | 408 | 409 | 410 | 411 | 412 | 413 | 414 | 415 | 416 | 417 | 418 | 419 | 420 | 421 | 422 | 423 | 424 | 426 | 428 | 429 | 431 | 451;
export type HTTPStatusCode5xx = 500 | 501 | 502 | 503 | 504 | 505 | 507 | 511;
export type HTTPStatusCodes = HTTPStatusCode1xx | HTTPStatusCode2xx | HTTPStatusCode3xx | HTTPStatusCode4xx | HTTPStatusCode5xx;




export type accessPersonaListResponse200 = {
  data: DemoPersonaList
  status: 200
}

export type accessPersonaListResponseDefault = {
  data: ProblemResponse
  status: Exclude<HTTPStatusCodes, 200>
}

export type accessPersonaListResponseSuccess = (accessPersonaListResponse200) & {
  headers: Headers;
};
export type accessPersonaListResponseError = (accessPersonaListResponseDefault) & {
  headers: Headers;
};

export const getAccessPersonaListUrl = () => {




  return `/api/v1/auth/personas`
}

/**
 * Демо-трек (эпик 08, заметка): экран входа сначала предлагает выбрать демо-персону — псевдоним из стартовой политики (normative/policy) с ролью и областью. Вне демо-профиля операция отвечает 404 (api.not_found), и экран показывает только вход по логину.
 * @summary Демо-персоны для входа без пароля
 */
export const accessPersonaList = async ( options?: RequestInit): Promise<accessPersonaListResponseSuccess> => {

  const res = await fetch(getAccessPersonaListUrl(),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: accessPersonaListResponseError['data'], status?: number} = new globalThis.Error();
    const data : accessPersonaListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: accessPersonaListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as accessPersonaListResponseSuccess
}





export const getAccessPersonaListQueryKey = () => {
    return [
    'api','v1','auth','personas'
    ] as const;
    }


export const getAccessPersonaListQueryOptions = <TData = Awaited<ReturnType<typeof accessPersonaList>>, TError = globalThis.Error & { info?: ProblemResponse; status?: number }>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof accessPersonaList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAccessPersonaListQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof accessPersonaList>>> = ({ signal }) => accessPersonaList({ signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof accessPersonaList>>, TError, TData>
}

export type AccessPersonaListQueryResult = NonNullable<Awaited<ReturnType<typeof accessPersonaList>>>
export type AccessPersonaListQueryError = globalThis.Error & { info?: ProblemResponse; status?: number }


/**
 * @summary Демо-персоны для входа без пароля
 */

export function useAccessPersonaList<TData = Awaited<ReturnType<typeof accessPersonaList>>, TError = globalThis.Error & { info?: ProblemResponse; status?: number }>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof accessPersonaList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAccessPersonaListQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type accessSessionReadResponse200 = {
  data: Session
  status: 200
}

export type accessSessionReadResponseDefault = {
  data: ProblemResponse
  status: Exclude<HTTPStatusCodes, 200>
}

export type accessSessionReadResponseSuccess = (accessSessionReadResponse200) & {
  headers: Headers;
};
export type accessSessionReadResponseError = (accessSessionReadResponseDefault) & {
  headers: Headers;
};

export const getAccessSessionReadUrl = () => {




  return `/api/v1/auth/session`
}

/**
 * Пользователь, активная роль, область, смена, рабочее место; 401 access.unauthenticated — сеанса нет.
 * @summary Текущий сеанс
 */
export const accessSessionRead = async ( options?: RequestInit): Promise<accessSessionReadResponseSuccess> => {

  const res = await fetch(getAccessSessionReadUrl(),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: accessSessionReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : accessSessionReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: accessSessionReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as accessSessionReadResponseSuccess
}





export const getAccessSessionReadQueryKey = () => {
    return [
    'api','v1','auth','session'
    ] as const;
    }


export const getAccessSessionReadQueryOptions = <TData = Awaited<ReturnType<typeof accessSessionRead>>, TError = globalThis.Error & { info?: ProblemResponse; status?: number }>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof accessSessionRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAccessSessionReadQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof accessSessionRead>>> = ({ signal }) => accessSessionRead({ signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof accessSessionRead>>, TError, TData>
}

export type AccessSessionReadQueryResult = NonNullable<Awaited<ReturnType<typeof accessSessionRead>>>
export type AccessSessionReadQueryError = globalThis.Error & { info?: ProblemResponse; status?: number }


/**
 * @summary Текущий сеанс
 */

export function useAccessSessionRead<TData = Awaited<ReturnType<typeof accessSessionRead>>, TError = globalThis.Error & { info?: ProblemResponse; status?: number }>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof accessSessionRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAccessSessionReadQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type accessSessionCreateResponse201 = {
  data: Session
  status: 201
}

export type accessSessionCreateResponseDefault = {
  data: ProblemResponse
  status: Exclude<HTTPStatusCodes, 201>
}

export type accessSessionCreateResponseSuccess = (accessSessionCreateResponse201) & {
  headers: Headers;
};
export type accessSessionCreateResponseError = (accessSessionCreateResponseDefault) & {
  headers: Headers;
};

export const getAccessSessionCreateUrl = () => {




  return `/api/v1/auth/session`
}

/**
 * FR-128. Вход демо-персоной (persona_id) или по логину. Пароль пока необязателен (демо-трек); после эпика 08 — обязателен для входа по логину (сеанс scs, argon2id). Ответ ставит cookie сеанса.
 * @summary Войти
 */
export const accessSessionCreate = async (sessionCreate: SessionCreate, options?: RequestInit): Promise<accessSessionCreateResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getAccessSessionCreateUrl(),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(sessionCreate)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: accessSessionCreateResponseError['data'], status?: number} = new globalThis.Error();
    const data : accessSessionCreateResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: accessSessionCreateResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as accessSessionCreateResponseSuccess
}





export const getAccessSessionCreateMutationKey = () => ['accessSessionCreate'] as const;

export const getAccessSessionCreateMutationOptions = <TError = globalThis.Error & { info?: ProblemResponse; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof accessSessionCreate>>, TError,AccessSessionCreateMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof accessSessionCreate>>, TError,AccessSessionCreateMutationVariables, TContext> => {

const mutationKey = getAccessSessionCreateMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof accessSessionCreate>>, AccessSessionCreateMutationVariables> = (props) => {
          const {data} = props ?? {};

          return  accessSessionCreate(data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type AccessSessionCreateMutationResult = NonNullable<Awaited<ReturnType<typeof accessSessionCreate>>>
    export type AccessSessionCreateMutationBody = SessionCreate
    export type AccessSessionCreateMutationError = globalThis.Error & { info?: ProblemResponse; status?: number }
    export type AccessSessionCreateMutationVariables = {data: SessionCreate}

    /**
 * @summary Войти
 */
export const useAccessSessionCreate = <TError = globalThis.Error & { info?: ProblemResponse; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof accessSessionCreate>>, TError,AccessSessionCreateMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof accessSessionCreate>>,
        TError,
        AccessSessionCreateMutationVariables,
        TContext
      > => {
      return useMutation(getAccessSessionCreateMutationOptions(options), queryClient);
    }

export type accessSessionDeleteResponse204 = {
  data: void
  status: 204
}

export type accessSessionDeleteResponseDefault = {
  data: ProblemResponse
  status: Exclude<HTTPStatusCodes, 204>
}

export type accessSessionDeleteResponseSuccess = (accessSessionDeleteResponse204) & {
  headers: Headers;
};
export type accessSessionDeleteResponseError = (accessSessionDeleteResponseDefault) & {
  headers: Headers;
};

export const getAccessSessionDeleteUrl = () => {




  return `/api/v1/auth/session`
}

/**
 * @summary Выйти
 */
export const accessSessionDelete = async ( options?: RequestInit): Promise<accessSessionDeleteResponseSuccess> => {

  const res = await fetch(getAccessSessionDeleteUrl(),
  {
    ...options,
    method: 'DELETE'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: accessSessionDeleteResponseError['data'], status?: number} = new globalThis.Error();
    const data : accessSessionDeleteResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: accessSessionDeleteResponseSuccess['data'] = body ? JSON.parse(body) : undefined
  return { data, status: res.status, headers: res.headers } as accessSessionDeleteResponseSuccess
}





export const getAccessSessionDeleteMutationKey = () => ['accessSessionDelete'] as const;

export const getAccessSessionDeleteMutationOptions = <TError = globalThis.Error & { info?: ProblemResponse; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof accessSessionDelete>>, TError,void, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof accessSessionDelete>>, TError,void, TContext> => {

const mutationKey = getAccessSessionDeleteMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof accessSessionDelete>>, void> = () => {


          return  accessSessionDelete(fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type AccessSessionDeleteMutationResult = NonNullable<Awaited<ReturnType<typeof accessSessionDelete>>>

    export type AccessSessionDeleteMutationError = globalThis.Error & { info?: ProblemResponse; status?: number }


    /**
 * @summary Выйти
 */
export const useAccessSessionDelete = <TError = globalThis.Error & { info?: ProblemResponse; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof accessSessionDelete>>, TError,void, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof accessSessionDelete>>,
        TError,
        void,
        TContext
      > => {
      return useMutation(getAccessSessionDeleteMutationOptions(options), queryClient);
    }

export type accessDeskReadResponse200 = {
  data: Desk
  status: 200
}

export type accessDeskReadResponseDefault = {
  data: ProblemResponse
  status: Exclude<HTTPStatusCodes, 200>
}

export type accessDeskReadResponseSuccess = (accessDeskReadResponse200) & {
  headers: Headers;
};
export type accessDeskReadResponseError = (accessDeskReadResponseDefault) & {
  headers: Headers;
};

export const getAccessDeskReadUrl = () => {




  return `/api/v1/desk`
}

/**
 * AD-21: стол — данные normative/desks/‹роль›.yaml (раскладка → вкладки → слоты → виджеты → срез и плотность), отдаётся через access.Queries.Desks. Для роли-наследника без своего файла — стол ближайшей базовой роли (inherits).
 * @summary Стол активной роли
 */
export const accessDeskRead = async ( options?: RequestInit): Promise<accessDeskReadResponseSuccess> => {

  const res = await fetch(getAccessDeskReadUrl(),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: accessDeskReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : accessDeskReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: accessDeskReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as accessDeskReadResponseSuccess
}





export const getAccessDeskReadQueryKey = () => {
    return [
    'api','v1','desk'
    ] as const;
    }


export const getAccessDeskReadQueryOptions = <TData = Awaited<ReturnType<typeof accessDeskRead>>, TError = globalThis.Error & { info?: ProblemResponse; status?: number }>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof accessDeskRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAccessDeskReadQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof accessDeskRead>>> = ({ signal }) => accessDeskRead({ signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof accessDeskRead>>, TError, TData>
}

export type AccessDeskReadQueryResult = NonNullable<Awaited<ReturnType<typeof accessDeskRead>>>
export type AccessDeskReadQueryError = globalThis.Error & { info?: ProblemResponse; status?: number }


/**
 * @summary Стол активной роли
 */

export function useAccessDeskRead<TData = Awaited<ReturnType<typeof accessDeskRead>>, TError = globalThis.Error & { info?: ProblemResponse; status?: number }>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof accessDeskRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAccessDeskReadQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type accessPermissionListResponse200 = {
  data: PermissionList
  status: 200
}

export type accessPermissionListResponseDefault = {
  data: ProblemResponse
  status: Exclude<HTTPStatusCodes, 200>
}

export type accessPermissionListResponseSuccess = (accessPermissionListResponse200) & {
  headers: Headers;
};
export type accessPermissionListResponseError = (accessPermissionListResponseDefault) & {
  headers: Headers;
};

export const getAccessPermissionListUrl = (params?: AccessPermissionListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/permissions?${stringifiedParams}` : `/api/v1/permissions`
}

/**
 * AD-15 «Для фронтенда»: без subject — плоский список «действие → объект» для @casl/vue; с subject и id — допустимые действия по конкретному объекту. Сервер вычисляет его тем же Enforce, что проверяет команды («в списке ⇔ разрешено»). На момент as_of (воспроизведение) команд в списке нет.
 * @summary Разрешённые действия
 */
export const accessPermissionList = async (params?: AccessPermissionListParams, options?: RequestInit): Promise<accessPermissionListResponseSuccess> => {

  const res = await fetch(getAccessPermissionListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: accessPermissionListResponseError['data'], status?: number} = new globalThis.Error();
    const data : accessPermissionListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: accessPermissionListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as accessPermissionListResponseSuccess
}





export const getAccessPermissionListQueryKey = (params?: MaybeRefOrGetter<AccessPermissionListParams>,) => {
    return [
    'api','v1','permissions', ...(params ? [params] : [])
    ] as const;
    }


export const getAccessPermissionListQueryOptions = <TData = Awaited<ReturnType<typeof accessPermissionList>>, TError = globalThis.Error & { info?: ProblemResponse; status?: number }>(params?: MaybeRefOrGetter<AccessPermissionListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof accessPermissionList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAccessPermissionListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof accessPermissionList>>> = ({ signal }) => accessPermissionList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof accessPermissionList>>, TError, TData>
}

export type AccessPermissionListQueryResult = NonNullable<Awaited<ReturnType<typeof accessPermissionList>>>
export type AccessPermissionListQueryError = globalThis.Error & { info?: ProblemResponse; status?: number }


/**
 * @summary Разрешённые действия
 */

export function useAccessPermissionList<TData = Awaited<ReturnType<typeof accessPermissionList>>, TError = globalThis.Error & { info?: ProblemResponse; status?: number }>(
 params?: MaybeRefOrGetter<AccessPermissionListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof accessPermissionList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAccessPermissionListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type securityIntegrityReadResponse200 = {
  data: IntegrityStatus
  status: 200
}

export type securityIntegrityReadResponseDefault = {
  data: ProblemResponse
  status: Exclude<HTTPStatusCodes, 200>
}

export type securityIntegrityReadResponseSuccess = (securityIntegrityReadResponse200) & {
  headers: Headers;
};
export type securityIntegrityReadResponseError = (securityIntegrityReadResponseDefault) & {
  headers: Headers;
};

export const getSecurityIntegrityReadUrl = () => {




  return `/api/v1/integrity`
}

/**
 * AD-46: ant забирает последний подписанный отчёт верификатора у хранителя и журналирует security.integrity.checked. Индикатор на столах помечен «по данным сервера» и желтеет сам, если свежего отчёта нет дольше двух интервалов.
 * @summary Состояние целостности журнала «по данным сервера»
 */
export const securityIntegrityRead = async ( options?: RequestInit): Promise<securityIntegrityReadResponseSuccess> => {

  const res = await fetch(getSecurityIntegrityReadUrl(),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: securityIntegrityReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : securityIntegrityReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: securityIntegrityReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as securityIntegrityReadResponseSuccess
}





export const getSecurityIntegrityReadQueryKey = () => {
    return [
    'api','v1','integrity'
    ] as const;
    }


export const getSecurityIntegrityReadQueryOptions = <TData = Awaited<ReturnType<typeof securityIntegrityRead>>, TError = globalThis.Error & { info?: ProblemResponse; status?: number }>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof securityIntegrityRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getSecurityIntegrityReadQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof securityIntegrityRead>>> = ({ signal }) => securityIntegrityRead({ signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof securityIntegrityRead>>, TError, TData>
}

export type SecurityIntegrityReadQueryResult = NonNullable<Awaited<ReturnType<typeof securityIntegrityRead>>>
export type SecurityIntegrityReadQueryError = globalThis.Error & { info?: ProblemResponse; status?: number }


/**
 * @summary Состояние целостности журнала «по данным сервера»
 */

export function useSecurityIntegrityRead<TData = Awaited<ReturnType<typeof securityIntegrityRead>>, TError = globalThis.Error & { info?: ProblemResponse; status?: number }>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof securityIntegrityRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getSecurityIntegrityReadQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type notificationsSummaryReadResponse200 = {
  data: NotificationSummary
  status: 200
}

export type notificationsSummaryReadResponseDefault = {
  data: ProblemResponse
  status: Exclude<HTTPStatusCodes, 200>
}

export type notificationsSummaryReadResponseSuccess = (notificationsSummaryReadResponse200) & {
  headers: Headers;
};
export type notificationsSummaryReadResponseError = (notificationsSummaryReadResponseDefault) & {
  headers: Headers;
};

export const getNotificationsSummaryReadUrl = (params?: NotificationsSummaryReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/notifications/summary?${stringifiedParams}` : `/api/v1/notifications/summary`
}

/**
 * @summary Сводка уведомлений для шапки
 */
export const notificationsSummaryRead = async (params?: NotificationsSummaryReadParams, options?: RequestInit): Promise<notificationsSummaryReadResponseSuccess> => {

  const res = await fetch(getNotificationsSummaryReadUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: notificationsSummaryReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : notificationsSummaryReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: notificationsSummaryReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as notificationsSummaryReadResponseSuccess
}





export const getNotificationsSummaryReadQueryKey = (params?: MaybeRefOrGetter<NotificationsSummaryReadParams>,) => {
    return [
    'api','v1','notifications','summary', ...(params ? [params] : [])
    ] as const;
    }


export const getNotificationsSummaryReadQueryOptions = <TData = Awaited<ReturnType<typeof notificationsSummaryRead>>, TError = globalThis.Error & { info?: ProblemResponse; status?: number }>(params?: MaybeRefOrGetter<NotificationsSummaryReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof notificationsSummaryRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getNotificationsSummaryReadQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof notificationsSummaryRead>>> = ({ signal }) => notificationsSummaryRead(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof notificationsSummaryRead>>, TError, TData>
}

export type NotificationsSummaryReadQueryResult = NonNullable<Awaited<ReturnType<typeof notificationsSummaryRead>>>
export type NotificationsSummaryReadQueryError = globalThis.Error & { info?: ProblemResponse; status?: number }


/**
 * @summary Сводка уведомлений для шапки
 */

export function useNotificationsSummaryRead<TData = Awaited<ReturnType<typeof notificationsSummaryRead>>, TError = globalThis.Error & { info?: ProblemResponse; status?: number }>(
 params?: MaybeRefOrGetter<NotificationsSummaryReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof notificationsSummaryRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getNotificationsSummaryReadQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type itemItemLookupResponse200 = {
  data: ItemLookup
  status: 200
}

export type itemItemLookupResponseDefault = {
  data: ProblemResponse
  status: Exclude<HTTPStatusCodes, 200>
}

export type itemItemLookupResponseSuccess = (itemItemLookupResponse200) & {
  headers: Headers;
};
export type itemItemLookupResponseError = (itemItemLookupResponseDefault) & {
  headers: Headers;
};

export const getItemItemLookupUrl = (params: ItemItemLookupParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/items/lookup?${stringifiedParams}` : `/api/v1/items/lookup`
}

/**
 * Разрешение носителя (AD-41) → изделие. Не найдено — 404 api.not_found.
 * @summary Найти изделие по номеру детали или скану DataMatrix
 */
export const itemItemLookup = async (params: ItemItemLookupParams, options?: RequestInit): Promise<itemItemLookupResponseSuccess> => {

  const res = await fetch(getItemItemLookupUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: itemItemLookupResponseError['data'], status?: number} = new globalThis.Error();
    const data : itemItemLookupResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: itemItemLookupResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as itemItemLookupResponseSuccess
}





export const getItemItemLookupQueryKey = (params?: MaybeRefOrGetter<ItemItemLookupParams>,) => {
    return [
    'api','v1','items','lookup', ...(params ? [params] : [])
    ] as const;
    }


export const getItemItemLookupQueryOptions = <TData = Awaited<ReturnType<typeof itemItemLookup>>, TError = globalThis.Error & { info?: ProblemResponse; status?: number }>(params: MaybeRefOrGetter<ItemItemLookupParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof itemItemLookup>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getItemItemLookupQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof itemItemLookup>>> = ({ signal }) => itemItemLookup(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof itemItemLookup>>, TError, TData>
}

export type ItemItemLookupQueryResult = NonNullable<Awaited<ReturnType<typeof itemItemLookup>>>
export type ItemItemLookupQueryError = globalThis.Error & { info?: ProblemResponse; status?: number }


/**
 * @summary Найти изделие по номеру детали или скану DataMatrix
 */

export function useItemItemLookup<TData = Awaited<ReturnType<typeof itemItemLookup>>, TError = globalThis.Error & { info?: ProblemResponse; status?: number }>(
 params: MaybeRefOrGetter<ItemItemLookupParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof itemItemLookup>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getItemItemLookupQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type journalStreamSubscribeResponse200 = {
  data: EntityChanged
  status: 200
}

export type journalStreamSubscribeResponseDefault = {
  data: ProblemResponse
  status: Exclude<HTTPStatusCodes, 200>
}

export type journalStreamSubscribeResponseSuccess = (journalStreamSubscribeResponse200) & {
  headers: Headers;
};
export type journalStreamSubscribeResponseError = (journalStreamSubscribeResponseDefault) & {
  headers: Headers;
};

export const getJournalStreamSubscribeUrl = () => {




  return `/api/v1/stream`
}

/**
 * Канал `sse` из contracts/events/asyncapi.yaml: `id:` = seq, `event: entity_changed`, `data:` — EntityChanged. Фронтенд инвалидирует ключ Vue Query [сущность, id]; переподключение — с Last-Event-ID. Клиент — ручной модуль src/shared/api/sse (единственное исключение AD-20), типы — отсюда.
 * @summary Живые обновления столов (SSE)
 */
export const journalStreamSubscribe = async ( options?: RequestInit): Promise<journalStreamSubscribeResponseSuccess> => {

  const res = await fetch(getJournalStreamSubscribeUrl(),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: journalStreamSubscribeResponseError['data'], status?: number} = new globalThis.Error();
    const data : journalStreamSubscribeResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: journalStreamSubscribeResponseSuccess['data'] = body !== null ? body : ''
  return { data, status: res.status, headers: res.headers } as journalStreamSubscribeResponseSuccess
}





export const getJournalStreamSubscribeQueryKey = () => {
    return [
    'api','v1','stream'
    ] as const;
    }


export const getJournalStreamSubscribeQueryOptions = <TData = Awaited<ReturnType<typeof journalStreamSubscribe>>, TError = globalThis.Error & { info?: ProblemResponse; status?: number }>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof journalStreamSubscribe>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getJournalStreamSubscribeQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof journalStreamSubscribe>>> = ({ signal }) => journalStreamSubscribe({ signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof journalStreamSubscribe>>, TError, TData>
}

export type JournalStreamSubscribeQueryResult = NonNullable<Awaited<ReturnType<typeof journalStreamSubscribe>>>
export type JournalStreamSubscribeQueryError = globalThis.Error & { info?: ProblemResponse; status?: number }


/**
 * @summary Живые обновления столов (SSE)
 */

export function useJournalStreamSubscribe<TData = Awaited<ReturnType<typeof journalStreamSubscribe>>, TError = globalThis.Error & { info?: ProblemResponse; status?: number }>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof journalStreamSubscribe>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getJournalStreamSubscribeQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







