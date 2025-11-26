/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React from 'react';
import CardPro from '../../common/ui/CardPro';
import ErrorLogsTable from './ErrorLogsTable';
import ErrorLogsActions from './ErrorLogsActions';
import ErrorLogsFilters from './ErrorLogsFilters';
import ColumnSelectorModal from './modals/ColumnSelectorModal';
import DetailModal from './modals/DetailModal';
import { useErrorLogsData } from '../../../hooks/error-logs/useErrorLogsData';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import { createCardProPagination } from '../../../helpers/utils';

const ErrorLogsPage = () => {
  const logsData = useErrorLogsData();
  const isMobile = useIsMobile();

  return (
    <>
      {/* Modals */}
      <ColumnSelectorModal {...logsData} />
      <DetailModal {...logsData} />

      {/* Main Content */}
      <CardPro
        type='type2'
        statsArea={<ErrorLogsActions {...logsData} />}
        searchArea={<ErrorLogsFilters {...logsData} />}
        paginationArea={createCardProPagination({
          currentPage: logsData.activePage,
          pageSize: logsData.pageSize,
          total: logsData.logCount,
          onPageChange: logsData.handlePageChange,
          onPageSizeChange: logsData.handlePageSizeChange,
          isMobile: isMobile,
          t: logsData.t,
        })}
        t={logsData.t}
      >
        <ErrorLogsTable {...logsData} />
      </CardPro>
    </>
  );
};

export default ErrorLogsPage;
