// СГЕНЕРИРОВАНО frontend/scripts/generate.mjs — руками не править (AD-20).
// Источник: contracts/statuses.yaml

/** Палитра тонов: цвет — только для статуса и главного действия (NFR-UI-2). */
export const statusPalette = {
  "neutral": "#8A8F98",
  "info": "#2F6FDB",
  "attention": "#E0A100",
  "danger": "#D64545",
  "critical": "#8B1E1E",
  "success": "#2E9E5B",
  "qualified": "#7A9A2E",
  "muted": "#B8BDC6"
} as const

/** Тон цвета статуса. */
export type StatusTone = keyof typeof statusPalette

/** Оси статуса изделия (PRD §3b): у каждой один модуль-владелец. */
export const statusAxes = {
  "position": {
    "title": "Положение в процессе",
    "owner": "process",
    "values": {
      "in_queue": {
        "label": "В очереди",
        "tone": "neutral"
      },
      "in_progress": {
        "label": "В работе",
        "tone": "info"
      },
      "at_inspection": {
        "label": "На контроле",
        "tone": "info"
      },
      "at_presentation_point": {
        "label": "На точке предъявления",
        "tone": "attention"
      },
      "in_transit": {
        "label": "В перемещении",
        "tone": "info"
      },
      "in_storage": {
        "label": "На хранении",
        "tone": "neutral"
      },
      "isolated": {
        "label": "В изоляции",
        "tone": "danger"
      },
      "completed": {
        "label": "Завершено",
        "tone": "muted"
      }
    }
  },
  "quality": {
    "title": "Состояние качества",
    "owner": "quality",
    "values": {
      "not_inspected": {
        "label": "Не проверено",
        "tone": "neutral"
      },
      "conforming": {
        "label": "Годно",
        "tone": "success"
      },
      "accepted_with_concession": {
        "label": "Годно по разрешению на отклонение",
        "tone": "qualified"
      },
      "unable_to_assess": {
        "label": "Оценка невозможна",
        "tone": "attention"
      },
      "signal": {
        "label": "Сигнал о признаке дефекта",
        "tone": "attention"
      },
      "nonconforming": {
        "label": "Несоответствие подтверждено",
        "tone": "danger"
      }
    }
  },
  "disposition": {
    "title": "Решение по изделию",
    "owner": "nonconformity",
    "values": {
      "none": {
        "label": "Решения нет",
        "tone": "neutral"
      },
      "rework": {
        "label": "Переделка",
        "tone": "attention"
      },
      "repair": {
        "label": "Ремонт",
        "tone": "attention"
      },
      "use_as_is": {
        "label": "Как есть",
        "tone": "qualified"
      },
      "scrap": {
        "label": "Списание",
        "tone": "critical"
      },
      "return_to_supplier": {
        "label": "Возврат поставщику",
        "tone": "critical"
      }
    }
  },
  "containment": {
    "title": "Сдерживание",
    "owner": "nonconformity",
    "values": {
      "none": {
        "label": "Без сдерживания",
        "tone": "neutral"
      },
      "observe": {
        "label": "Наблюдать",
        "tone": "info"
      },
      "additional_check": {
        "label": "Доп. проверка",
        "tone": "attention"
      },
      "item_hold": {
        "label": "Блок изделия",
        "tone": "danger"
      },
      "lot_hold": {
        "label": "Блок партии",
        "tone": "danger"
      }
    }
  },
  "erp_accounting": {
    "title": "Учёт в 1С",
    "owner": "erp",
    "values": {
      "not_sent": {
        "label": "Не передано в 1С",
        "tone": "neutral"
      },
      "accepted_into_work": {
        "label": "Принято в работу",
        "tone": "info"
      },
      "moved": {
        "label": "Перемещено",
        "tone": "info"
      },
      "transferred_to_scrap": {
        "label": "Переведено в брак",
        "tone": "critical"
      },
      "returned_to_supplier": {
        "label": "Возвращено поставщику",
        "tone": "critical"
      },
      "released": {
        "label": "Выпущено",
        "tone": "success"
      }
    }
  },
  "incident": {
    "title": "Статус в инциденте (для каждого инцидента отдельно)",
    "owner": "analysis",
    "values": {
      "confirmed": {
        "label": "Подтверждено",
        "tone": "danger"
      },
      "suspect": {
        "label": "Под подозрением",
        "tone": "attention"
      },
      "excluded": {
        "label": "Исключено",
        "tone": "success"
      },
      "unknown": {
        "label": "Неизвестно",
        "tone": "neutral"
      }
    }
  }
} as const

/** Ось статуса. */
export type StatusAxis = keyof typeof statusAxes

/** Дополнительные словари — не оси, но едины для всех экранов. */
export const statusDictionaries = {
  "incident_action": {
    "title": "Что делать с изделием в инциденте (FR-62)",
    "values": {
      "observe": {
        "label": "Наблюдать",
        "tone": "info"
      },
      "check": {
        "label": "Проверить",
        "tone": "attention"
      },
      "block": {
        "label": "Заблокировать",
        "tone": "danger"
      },
      "release": {
        "label": "Выпустить",
        "tone": "success"
      }
    }
  },
  "process_containment": {
    "title": "Сдерживание процесса — отдельный объект от сдерживания изделий (FR-49)",
    "values": {
      "process_point_stop": {
        "label": "Стоп точки процесса",
        "tone": "danger"
      },
      "critical_stop": {
        "label": "Критическая остановка",
        "tone": "critical"
      }
    }
  },
  "erp_message_status": {
    "title": "Состояние исходящего учётного сообщения (не ось: ось меняет только квитанция)",
    "values": {
      "queued": {
        "label": "В очереди на отправку",
        "tone": "neutral"
      },
      "sent": {
        "label": "Отправлено — ждём подтверждения 1С",
        "tone": "attention"
      },
      "acknowledged": {
        "label": "Подтверждено 1С",
        "tone": "success"
      },
      "rejected": {
        "label": "Ошибка обмена",
        "tone": "danger"
      },
      "quarantined": {
        "label": "В карантине — нужна переотправка",
        "tone": "danger"
      }
    }
  },
  "investigation_stage": {
    "title": "Стадия расследования инцидента (стол технолога): область → гипотезы → причина → меры → проверка эффективности → закрыто",
    "values": {
      "scope_defined": {
        "label": "Область определена",
        "tone": "info"
      },
      "hypothesis": {
        "label": "Проверка гипотез",
        "tone": "attention"
      },
      "cause_confirmed": {
        "label": "Причина установлена",
        "tone": "info"
      },
      "action_assigned": {
        "label": "Меры назначены",
        "tone": "info"
      },
      "effectiveness_check": {
        "label": "Проверка эффективности",
        "tone": "attention"
      },
      "closed": {
        "label": "Расследование закрыто",
        "tone": "muted"
      }
    }
  },
  "item_summary": {
    "title": "Сводный статус изделия для списков и карты (словарь продукта «Статусы изделия»); вычисляется из шести осей, сам осью не является",
    "values": {
      "in_process": {
        "label": "В работе",
        "tone": "info"
      },
      "suspect": {
        "label": "Под подозрением",
        "tone": "attention"
      },
      "reinspection_required": {
        "label": "Ожидает доп. контроля",
        "tone": "attention"
      },
      "hold": {
        "label": "Заблокировано",
        "tone": "danger"
      },
      "pending_decision": {
        "label": "Ожидает решения",
        "tone": "attention"
      },
      "nonconforming": {
        "label": "Несоответствие подтверждено",
        "tone": "danger"
      },
      "cleared": {
        "label": "Исключено из подозрения",
        "tone": "success"
      },
      "released": {
        "label": "Разрешено к движению",
        "tone": "success"
      },
      "in_rework": {
        "label": "На переделке",
        "tone": "attention"
      },
      "in_repair": {
        "label": "На ремонте",
        "tone": "attention"
      },
      "accepted_with_concession": {
        "label": "Годно по разрешению на отклонение",
        "tone": "qualified"
      },
      "scrapped": {
        "label": "Списано",
        "tone": "critical"
      },
      "returned": {
        "label": "Возвращено поставщику",
        "tone": "critical"
      }
    }
  }
} as const
