/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { PostItem } from './postItem';
import type { PostPerson } from './postPerson';
import type { PostRowPresence } from './postRowPresence';

export interface PostRow {
  /** Нет — никто не назначен. */
  assigned?: PostPerson;
  /** Нет — на посту нет изделия. */
  current_item?: PostItem;
  /** На месте; по СКУД на месте, но ключ не вставлен; ключ вставлен, а владельца нет в зоне; нет ни в зоне, ни ключа; никто не назначен; неизвестно. */
  presence: PostRowPresence;
  /** Участок (пост) — подпись. */
  station: string;
  workplace_id: string;
  /** Цех (FR-130). */
  workshop?: string;
  /** Имя цеха из справочника мест; нет — показывать код workshop. */
  workshop_name?: string;
}
