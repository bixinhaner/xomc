<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>eNB Monitor</title>
    
    <script type="text/javascript" src="${ctx}/js/element/Sortable.min.js?_=${omc_ver}"></script>
    <script type="text/javascript" src="${ctx}/js/element/vuedraggable.umd.min.js?_=${omc_ver}"></script>

    <style>
        .mairgin-left-30 .el-checkbox {
            margin-left: 30px;
        }
        .mairgin-left-30 .el-checkbox__label {
            font-size: 12px;
        }
        #cellInfo {
            border: 1px solid #DFE2EE;
		    border-radius: 10px;
            box-sizing: border-box;
            height: 100%;
            overflow: hidden;
        }
		.fixed-right-msg {
			display: flex;
			top: -32px;
			right: 100px;
			position: absolute;
            border: 1px solid #DFE2EE;
		    border-radius: 5px;
		}
        .fixed-right-msg .foldBtnCls{
            height: 26px;
            width: 26px;
            display: flex;
            flex-direction: column;
            justify-content: center;
            align-items: center;
			background-color: #fff;
            cursor: pointer;
            transform: rotate(90deg);
        }
        .collect-select .el-input {
            width: 120px;
        }
        
        #enbMonitor .el-ctable-toolbar,#gsmMonitor .el-ctable-toolbar{
            padding: 0px!important;
        }
        #cellInfo .flex-item-cls {
            height: 100%;
            overflow: auto;   
            position: relative;
            box-sizing: border-box;
        }
        .monitor-ctner .el-table th>.cell {
            white-space: nowrap;
        }
        .list-group-item:hover .sort-item-op {
            visibility: visible;
        }
        .list-group > span {
            display: flex;
            flex-direction: column;
            padding-left: 15px; 
            height: 450px;
            overflow: auto;
        }
        .list-group-item {
            display: inline-block;
            position: relative;
            padding: 0px 10px;
            margin: 3px 5px;
            width: 200px;
            border: 0px dashed #ddd;
            cursor: move;
        }
        .no-padding .slide-content {
            padding: 0px;
        }
        .no-border .el-card__body {
            border: none;
        }
        .cellNameClass{
            display: inline-block;
            height: 18px;
            width: 18px;
            text-align: center;
            line-height: 18px;
            border: 1px solid #DCDFE6;
            border-radius: 2px;
        }
        .cellNameClass .el-icon::before{
            font-size: 16px;
            color: #F2B354;
        }
        .cellNameClass:hover{
            border: 1px solid #4D84FF;
        }
        .mmeDetails {
            top: 30px;
            position:absolute;
            background:white;
            padding:10px 20px;
            box-shadow:4px 4px 19px 0 rgba(0,0,0,0.15);
            border:1px solid #d1ecf5;
            display:none;
            left:0;
            z-index:99999;
        }
        .syncNameInfo div{
            margin-right:20px;
            line-height:30px;
            
        }
        .activeStatusItem ,.inactiveStatusItem {
        	display:inline-block;
        }
        .activeStatusItem .el-icon,.inactiveStatusItem .el-icon{
            font-size:20px;
            vertical-align:bottom;
            margin-right:5px;
        }
        .activeStatusItem .el-icon-status-active:before{
            color:#67D972;
        }
        .inactiveStatusItem .el-icon-status-active:before{
            color:#E88282;
        }
        .mmeConnItem,.mmeDisconnItem,.rfConnItem,.rfDisconnItem {
            display: flex;
            white-space: nowrap;
            align-items: center;
        }
        .mmeConnItem .el-icon-status-MME:before,.mmeConnItem .el-icon-status-MME1:before,.mmeConnItem .el-icon-status-MME2:before{
            color:#67D972;
            font-size:18px;
        }
        .mmeDisconnItem .el-icon-status-MME:before,.mmeDisconnItem .el-icon-status-MME1:before,.mmeDisconnItem .el-icon-status-MME2:before{
            color:#E88282;
            font-size:18px;
        }
        .rfConnItem .el-icon-status-RF:before,.rfConnItem .el-icon-status-RF1:before,.rfConnItem .el-icon-status-RF2:before{
            color:#67D972;
            font-size:20px;
        }
        .rfDisconnItem .el-icon-status-RF:before,.rfDisconnItem .el-icon-status-RF1:before,.rfDisconnItem .el-icon-status-RF2:before{
            color:#E88282;
            font-size:20px;
        }
        
        .cpeLwaKai{
            display: inline-block;
            margin-top:2px;
            width:25px;
            height:25px;
            background:url(${ctx}/css/images/newIcon/statusIcon/TURBOkai.png) no-repeat;
        }
        .cpeLwaGuan{
            display: inline-block;
            margin-top:2px;
            width:25px;
            height:25px;
            background:url(${ctx}/css/images/newIcon/statusIcon/TURBOguan.png) no-repeat;
        }
        .cpeLwaWu{
            display: inline-block;
            margin-top:2px;
            width:25px;
            height:25px;
            background:url(${ctx}/css/images/newIcon/statusIcon/TURBOwu.png) no-repeat;
        }
        .showHideItem {
            position:absolute;
            background:white;
            z-index:888;
            width: 1080px;
            top:86px;
            left: 0px;
            display: none;
            box-shadow:5px 10px 23px 0px rgba(0,0,0,0.15);
        }
        .enbMonitorForm .selectItem .el-input{
            width:127px;
        }
        .enbMonitorForm .selectItem .el-input__inner{
            height:30px;
            border-radius:2px 0px 0px 2px;
        }
        .enbMonitorForm .queryGroup{
            height:28px;
            border-radius:0px 4px 4px 0px;
            margin-left:0px;
        }
        .enbMonitorForm .queryGroup .el-input__inner{
            height:28px;
        }
        .enbMonitorForm .el-form-item{
            display:inline-block;
            margin-bottom:0px;
        }
        .enbMonitorForm .selectContent{
            flex:1
        }
        .enbMonitorForm .selectContent .el-form-item{
            margin-bottom:6px;
        }
        .enbMonitorForm .selectContent .el-input{
            width:100px;
            border-radius:2px;
        }
        .enbMonitorForm .selectContent .el-input__inner{
            height:24px;
        }
        .enbMonitorForm .el-form-item__label{
            line-height:24px;
            text-align:right;
            font-size:12px;
        }
        .enbMonitorForm .el-tag__close{
            display:none;
        }
        .enbMonitorForm .el-select__tags{
            height:24px;
            overflow:hidden;
        }
        .enbMonitorForm .el-tag--small{
            height:16px;
            line-height:16px;
            background:#fff;
        }
        .enbMonitorForm .el-tag--small:nth-of-type(2){
            display:none;
        }
        .el-select-dropdown.is-multiple .el-select-dropdown__item.selected::after{
            right:2px;
        }

        .col-group {
            margin: 5px 10px;
            display: flex;
            flex-wrap: wrap;
        }
        .col-group .el-checkbox {
            min-width: 170px;
        }
        .col-group .el-checkbox__label {
            font-size: 12px;
        }

        .showHideItem input{
            margin-top:-2px;
            margin-bottom:1px;
            vertical-align:middle;
            margin-right:20px;
        }
        .selectAll{
            height:32px;
            width:334px;
            padding:28px 0px 0px 30px;
        }
        
        .showHideItem .select-all-cls {
            padding: 10px 0 0 9px;
            display: flex;
            align-items: center;
        }
        .select-all-cls > i {
            margin-right: 5px;
        }
        .showHideItem .select-all-cls > span {
            font-size: 14px;
            margin-left: 10px;
            color: #606266;
        }
        .showHideItem .el-icon-close1::before {
            color: #333;
        }
        .showHideItem #sortAndShowColumnBoxCls{
            height: 630px;
        }
        .showHideItem #sortAndShowColumnBoxCls .el-checkbox__input.is-disabled.is-checked .el-checkbox__inner{
		    opacity: 0.3!important;
        }
        .showHideItem #sortAndShowColumnBoxCls .list-group > span{
            padding-left: 0px;
            height: calc(100% - 52px);
            overflow-x: hidden;
        }
        .showHideItem #sortAndShowColumnBoxCls .list-group .el-checkbox__label{
            padding-left: 0px;
        }
        .showHideItem #sortAndShowColumnBoxCls .list-group .list-group-item {
            width: 260px;
        }
        .left-title-list li{
            height:36px;
            line-height:36px;
            padding:0 20px;
            cursor: pointer;
            border-bottom: 1px solid #EEE;
            word-break: keep-all;
        }
        .left-title-list li.active {
            color: #4D84FF;
            background-color: #EDF6FF;
        }

        /* 列表告警显示样式 */
        .alarmListSty{
            display:inline-block;
            min-width:13px;
            height:23px;
            padding:0 5px;
            line-height:24px;
            border-radius:23px;
            text-align:center;
            color:#FFFFFF;
            -webkit-transform:scale(0.8);
            font-size:12px;
            cursor:pointer;
        }
        #cellInfo .alarmCritical{
            background:#E88282;
        }
        #cellInfo .alarmMajor{
            background:#DCAA5E;
        }
        #cellInfo .alarmMinor{
            background:#CCCC66;
        }
        #cellInfo .alarmWarning{
            background:#9AF0FE;
        }

        .mme-list {
            position: relative;
            display: flex;
            max-width: 825px;
            flex-wrap: wrap;
            max-height: 500px;
            overflow: auto;
            padding:0 15px 15px 15px;
        }
        .mme-list-item {
            display: flex;
            padding: 10px;
            border: 1px solid #E9E9E9;
            margin:5px;
            border-radius:2px;
        }
        .mme-list-item .el-icon-status-MME {
        	margin-top:20px;
        }
        .specialPlmn { border-top: 1px solid #E9E9E9; }
        .objectPlmnItem { border-right: none; border-top: none; }
        .mme-info {
            display: flex;
            flex-direction: column;
            margin-left:5px;
        }

        .activeStatusItem .status-tip,
        .inactiveStatusItem .status-tip {
            display: none;
            position: absolute;
            z-index: 1000;
            padding: 2px 10px;
            background: #fff;
            border-radius: 2px;
            box-shadow: 2px 3px 10px #d1ecf5;
            top: 35px;
        }
        .activeStatusItem:hover .status-tip,
        .inactiveStatusItem:hover .status-tip {
            display: inline-block;
        }

        .el-table .el-table__row:last-child .status-tip {
            top: -25px;
        }
        .el-table .el-table__row:first-child .status-tip {
            top: 35px;
        }
        .warnTips {
        	font-size:18px;
        	vertical-align:middle;
        }
        .warnTips:before {
        	color:red;
        }
        .yellowIcon::before{
            color: #FF973E;
        }
        .greenIcon::before{
             color: #67D972;
        }
        .redIcon::before ,.redColor{
            color: #E88282;
        }
        .greenIcon::before,.redIcon::before,.yellowIcon::before{
            font-size: 20px;
        }
        .iconFlexCls{
            display: flex;
            align-items: center;
        }
        .cellActivePopoverClass{
            padding: 10px;
        }
        #activeCellDialog .borderCardItemCls{
            display: flex;
            padding: 5px 10px;
        }
        #activeCellDialog .borderCardItemLabelCls{
            width:90px;
            display:inline-block;
            font-size: 14px;
        }
        .mmePopoverClass .el-popover__title{
            height:35px;
            line-height:35px;
            margin-bottom:0;
            padding:0 20px 0;
        }
        .haloXIconBox{
            display: inline-block;
            height: 16px;
            width: 16px;
            line-height: 16px;
            text-align: center;
            font-size: 12px;
            font-weight: 600;
            color:#FFFFFF;
            background-color: #000;
        }
        .haloDModeCodeCls{
            display: inline-block;
            width: 16px;
            height: 16px;
            line-height: 16px;
            text-align: center;
            font-size: 12px;
            border-radius: 2px;
            color: #FFFFFF;
            background-color: #73B8FF;
            margin-left: 3px;
        }
        .tooltipCls.is-dark{
            background : #959595 ;
            color : #FFFFFF ;
        }
        .tooltipCls[x-placement^=top] .popper__arrow ,
        .tooltipCls[x-placement^=top] .popper__arrow::after{
            border-top-color: #959595!important;
        }

        .tooltipCls[x-placement^=bottom] .popper__arrow ,
        .tooltipCls[x-placement^=bottom] .popper__arrow::after {
            border-bottom-color: #959595!important;
        }
        .tooltipCls[x-placement^=right] .popper__arrow ,
        .tooltipCls[x-placement^=right] .popper__arrow::after {
            border-right-color: #959595!important;
        }
        .tooltipCls[x-placement^=left] .popper__arrow ,
        .tooltipCls[x-placement^=left] .popper__arrow::after {
            border-left-color: #959595!important;
        }
        .halodCellPopoverClass .buttonBoxCls{
            height: 28px;
            width: 28px;
            display: flex;
            align-items: center;
            justify-content: center;
            border:1px solid #DFE2EE;
            border-radius: 8px;
            margin-left: 10px
        }
        .halodCellPopoverClass .buttonBoxCls .el-icon::before{
            color: #7A7992;
            font-size: 16px;
        }
        .halodCellPopoverClass .buttonBoxCls:hover{
            background-color: #F5F7FA;
        }
        .halodCellPopoverClass .itemValueCls{
            display:inline-block;
            overflow:hidden;
            text-overflow:ellipsis;
            word-break:break-all;
            white-space: nowrap;
            
        }
        
        .need-reboot .el-form-item__content > .el-switch::after {
        	position: absolute;
        	bottom: -25px;
        	white-space: nowrap;
        }
        
        .need-reboot .el-form-item__content > .el-switch::after,
        .need-reboot .el-form-item__content > .el-select::after,
        .need-reboot .el-form-item__content > .el-input::after {
            content: 'Need reboot!';
            color: red;
            font-size: 12px;
        }
        
        .is-error.need-reboot .el-form-item__content > .el-switch::after,
        .is-error.need-reboot .el-form-item__content > .el-select::after,
        .is-error.need-reboot .el-form-item__content > .el-input::after {
            display: none;
        }
        .lockStyle:before { color: #FF9D47; }
        .lockStyle { margin-right:5px;font-size:14px; padding: 3px; border: 1px solid #DFE2EE; border-radius: 4px; }
        .lockStyle:hover, .lockStyle:active { border: 1px solid #4D84FF; }
        .lockReasonDialog .el-dialog__footer { padding: 10px !important; border-top: 1px solid #E9EDF9; }
        .infoTipPover { padding: 6px 10px !important; border-radius: 4px; border: 1px solid #E9E9E9; font-size: 12px; max-height: 300px; overflow: auto;}
        .apn-list-item { padding: 4px 8px 0;}
        .apn-info { font-size: 14px; padding-bottom: 2px;}
        .marginRight16 { color: #3D3D3D; }
        
        .hide-input input {
            display: none;
        }
        .el-icon-menu-maintenance.black-color::before {
            color: #333;
        }
        .settingSlide {
            width:70% !important;
            min-width:800px;
            left:auto;
        }
        .collect-bt {
            color:#4D84FF;
            margin-left: 15px;
            text-decoration: underline;
            cursor: pointer;
        }
        .status-statistic {
            display: flex;
            align-items: center;
            padding: 0 20px;
            box-sizing: border-box;
            color: #7A7992;
        }
        .status-statistic .el-icon::before {
            font-size: 16px;
            color: #4d84ff;
        }
        .status-statistic.active .el-icon::before {
            color: #fb9f50;
        }
        #tableHeadQuery .headQueryBoxInputWidthCls .queryGroup input {
            width: 340px;
        }

        .EnableClickBox {
            height: 20px;
            width: 50px;
            position: absolute;
            top:0px;
            left: 0px;
            right: 0px;
            bottom: 0px;
            margin: auto;
            z-index: 66;
            cursor: pointer;
            opacity: 0;
        }
        #cellInfo .remarkHeaderCls ,#cellInfo .remarkHeaderCls div{
            padding-left: unset;
            line-height: unset;
        }
        #cellInfo .remarkHeaderCls .el-input__suffix{
            line-height: 30px;
        }
        #cellInfo .remarkHeaderCls .el-input--suffix .el-input__inner{
            padding-right: 50px;
            box-sizing: border-box;
        }
        #cellInfo .remarkHeaderCls .el-icon::before{
            font-size: 16px;
            color: #7A7992;
        }
        /* 列配置悬浮层样式 */
        .column-config-popover {
            padding: 5px !important;
            max-height: 550px;
        }
        .column-config-popover .el-checkbox-group {
            display: flex;
            flex-wrap: wrap;
        }
        .column-config-popover .el-checkbox {
            margin-right: 10px;
            margin-bottom: 5px;
        }
        .flip-list-move {
            transition: transform 0.5s;
        }
        .no-move {
            transition: transform 0s;
        }
        .ghost {
            opacity: 0.5;
            background: #c8ebfb;
        }
    </style>
</head>
<body>
    <div id="cellInfo" class="monitor-ctner">
        <!-- tabs -->
        <el-tabs v-model="activeName" @tab-click="handleClick" class='newTabs' style="height: 100%;">
            <el-tab-pane label='<%=rb.getString("JiZhan")%>' name="eNB">
                <div id="enbMonitor" class="flex-item-cls" style="min-width: 1080px;display: flex; flex-direction: column; height: 100%;">
                    <div style="position: relative;background-color: #FFFFFF;">
                        <div class="toolbarHeadBtnBoxCls">
                            <div id="addDeviceOrImport" @click="showAddOrImport">
                                <el-popover trigger="click" placement="bottom-end" popper-class="monitorBtnPopperCls">
                                    <div class="el-card__header">
                                        <%=rb.getString("TianJiaJiZhan")%>
                                        <span style="font-size: 14px;" class="el-icon el-icon-close" onclick="document.body.click();"></span>
                                    </div>
                                    <div class="addOrImport-content" style="width: 700px; max-height: 500px;overflow: auto;"></div>
                                    <div slot="reference" class="newIconBoxCls-bt CODE_ENB_DEVICE_REGISTER hidden" style="right:56px;top:5px;" tip="<%=rb.getString("TianJia")%>">
                                        <span class="el-icon-plus el-icon"></span>
                                    </div>	
                                </el-popover>
                            </div>
                            <div id="export_op" @click="showExport">
                                <el-popover trigger="click" placement="bottom-end" popper-class="monitorBtnPopperCls">
                                    <div class="el-card__header">
                                        <%=rb.getString("DaoChu")%>
                                        <span style="color: 999;font-weight: normal;margin-left: 5px;">(<%=rb.getString("SuoYouCanShu")%>)</span>
                                        <span style="font-size: 14px;" class="el-icon el-icon-close" onclick="document.body.click();"></span>
                                    </div>
                                    <div id="enb_export_content" class="export-content" style="width: 920px; height: 500px;"></div>
                                    <div slot="reference" class="newIconBoxCls-bt" style="right:20px;top:5px;" tip="<%=rb.getString("DaoChu")%>">
                                        <span class="el-icon-operation-export el-icon"></span>
                                    </div>	
                                </el-popover>
                            </div>
                            <div v-show="enableCheckbox" class="selectBlukBoxCls">
                                <div class="selectMain">
                                    <div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable">
                                        <span class="el-icon-selected el-icon"></span>
                                        <span class="bulkSelectNumBoxCls">( {{selectedRows.length}} )</span>
                                    </div>
                                    <div class="selectTableBoxCls" style="position: absolute;top: 38px;left: 0px;" v-show="bulkSelectShow">
                                        <div class="selectBoxTitle">
                                            <span><%=rb.getString("YiXuan")%></span>
                                            <span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable"></span>
                                        </div>
                                        <div class="selectBoxMain">
                                            <div class="tableInfoCls">
                                                <div class="tableInfoHeader">
                                                    <div><%=rb.getString("Title_SheBeiBianMa")%></div>
                                                    <div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
                                                </div>
                                                <el-ctable 
                                                    id="bulkSelectTable" 
                                                    ref="bulkSelectTable" 
                                                    :data="selectedRows" 
                                                    :showHeader="false"
                                                    :rownumber="false"
                                                    :front-pagination="true"
                                                    height="300px" pagination="true" >
                                                    <el-table-column prop="id" v-if="false"></el-table-column>
                                                    <el-table-column width="588">
                                                        <template slot-scope="scope" >
                                                            <div class="tableItemCls">
                                                                <span>{{scope.row.serial_number}}</span>
                                                                <span @click="delBulkSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
                                                            </div>
                                                        </template>
                                                    </el-table-column>
                                                </el-ctable>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </div>
                            <div v-if="writableMap['CODE_ENB_MONITOR'] == true" :class="selectedRows.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="movecells">
                                <span class="el-icon el-icon-moveGroup"></span>
                                <span><%=rb.getString("YiDongDaoSheBeiZu")%></span>
                            </div>
                            <div v-if="writableMap['CODE_ENB_SYNCHRONIZE'] == true" :class="selectedRows.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="syncList">
                                <span class="el-icon el-icon-operation-synchronize"></span>
                                <span><%=rb.getString("TongBu")%></span>
                            </div>
                            <div v-if="writableMap['CODE_ENB_MONITOR'] == true && writableMap['CODE_ENB_REBOOT'] == true" :class="selectedRows.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="rebootList">
                                <span class="el-icon el-icon-operation-reboot"></span>
                                <span><%=rb.getString("ChongQi")%></span>
                            </div>
                            <div v-if="writableMap['CODE_ENB_MONITOR'] == true && writableMap['CODE_ENB_DEVICE_REGISTER'] == true" :class="selectedRows.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="recycleCells">
                                <span class="el-icon el-a-icon-Recyclebin"></span>
                                <span><%=rb.getString("HuiShouZhan")%></span>
                            </div>
                        </div>
                        <div id="toolbar_tableHomeCellList" style="position:relative;"></div>
                        <!-- 列配置按钮 -->
                        <el-popover 
                            ref="columnConfigPopover"
                            placement="right-start" 
                            width="980" 
                            trigger="click"
                            popper-class="column-config-popover"
                            v-model="columnConfigVisible">
                            <div style="display: flex; height: 480px;">
                                <!-- 左侧：列显示/隐藏选择 -->
                                <div style="flex: 3; padding: 0 10px; overflow: auto; border-right: 1px solid #e9e9e9;">
                                    <div style="font-weight: bold; padding: 10px 0;"><%=rb.getString("XuanZeLie")%></div>
                                    <!-- 全选 -->
                                    <div style="padding: 5px 10px;">
                                        <el-checkbox :indeterminate="isColumnIndeterminate" v-model="isColumnAllSelected" @change="handleColumnAllChange"><%=rb.getString("QuanXuan")%></el-checkbox>
                                    </div>
                                    <!-- 设备信息 -->
                                    <div>
                                        <div style="padding: 5px 10px; cursor: pointer;" @click="columnExpanded.device = !columnExpanded.device">
                                            <i :class="{'el-icon-arrow-down': columnExpanded.device, 'el-icon-arrow-right': !columnExpanded.device}"></i>
                                            <el-checkbox :indeterminate="columnForm.device.length > 0 && columnForm.device.length < filteredDeviceCol.length" v-model="isDeviceAllSelected" @change="handleDeviceAllChange">
                                                <span style="font-weight: bold;"><%=rb.getString("SheBeiXinXi")%></span>
                                            </el-checkbox>
                                        </div>
                                        <el-checkbox-group class="mairgin-left-30" v-show="columnExpanded.device" v-model="columnForm.device" style="padding-left: 30px;">
                                            <el-checkbox v-for="item in filteredDeviceCol" :label="item.code" :key="item.code" :disabled="item.disabled" style="min-width: 140px; margin-bottom: 5px;">{{item.label}}</el-checkbox>
                                        </el-checkbox-group>
                                    </div>
                                    <!-- 小区信息 -->
                                    <div>
                                        <div style="padding: 5px 10px; cursor: pointer;" @click="columnExpanded.cell = !columnExpanded.cell">
                                            <i :class="{'el-icon-arrow-down': columnExpanded.cell, 'el-icon-arrow-right': !columnExpanded.cell}"></i>
                                            <el-checkbox :indeterminate="columnForm.cell.length > 0 && columnForm.cell.length < filteredCellCol.length" v-model="isCellAllSelected" @change="handleCellAllChange">
                                                <span style="font-weight: bold;"><%=rb.getString("XiaoQuXinXi")%></span>
                                            </el-checkbox>
                                        </div>
                                        <el-checkbox-group class="mairgin-left-30" v-show="columnExpanded.cell" v-model="columnForm.cell" style="padding-left: 30px;">
                                            <el-checkbox v-for="item in filteredCellCol" :label="item.code" :key="item.code" :disabled="item.disabled" style="min-width: 140px; margin-bottom: 5px;">{{item.label}}</el-checkbox>
                                        </el-checkbox-group>
                                    </div>
                                    <!-- 状态 -->
                                    <div>
                                        <div style="padding: 5px 10px; cursor: pointer;" @click="columnExpanded.status = !columnExpanded.status">
                                            <i :class="{'el-icon-arrow-down': columnExpanded.status, 'el-icon-arrow-right': !columnExpanded.status}"></i>
                                            <el-checkbox :indeterminate="columnForm.status.length > 0 && columnForm.status.length < filteredStatusCol.length" v-model="isStatusAllSelected" @change="handleStatusAllChange">
                                                <span style="font-weight: bold;"><%=rb.getString("ZhuangTai")%></span>
                                            </el-checkbox>
                                        </div>
                                        <el-checkbox-group class="mairgin-left-30" v-show="columnExpanded.status" v-model="columnForm.status" style="padding-left: 30px;">
                                            <el-checkbox v-for="item in filteredStatusCol" :label="item.code" :key="item.code" :disabled="item.disabled" style="min-width: 140px; margin-bottom: 5px;">{{item.label}}</el-checkbox>
                                        </el-checkbox-group>
                                    </div>
                                    <!-- 网络设置 -->
                                    <div>
                                        <div style="padding: 5px 10px; cursor: pointer;" @click="columnExpanded.network = !columnExpanded.network">
                                            <i :class="{'el-icon-arrow-down': columnExpanded.network, 'el-icon-arrow-right': !columnExpanded.network}"></i>
                                            <el-checkbox :indeterminate="columnForm.network.length > 0 && columnForm.network.length < filteredNetworkCol.length" v-model="isNetworkAllSelected" @change="handleNetworkAllChange">
                                                <span style="font-weight: bold;"><%=rb.getString("WangLuoSheZhi")%></span>
                                            </el-checkbox>
                                        </div>
                                        <el-checkbox-group class="mairgin-left-30" v-show="columnExpanded.network" v-model="columnForm.network" style="padding-left: 30px;">
                                            <el-checkbox v-for="item in filteredNetworkCol" :label="item.code" :key="item.code" :disabled="item.disabled" style="min-width: 140px; margin-bottom: 5px;">{{item.label}}</el-checkbox>
                                        </el-checkbox-group>
                                    </div>
                                    <!-- 位置 -->
                                    <div>
                                        <div style="padding: 5px 10px; cursor: pointer;" @click="columnExpanded.location = !columnExpanded.location">
                                            <i :class="{'el-icon-arrow-down': columnExpanded.location, 'el-icon-arrow-right': !columnExpanded.location}"></i>
                                            <el-checkbox :indeterminate="columnForm.location.length > 0 && columnForm.location.length < filteredLocationCol.length" v-model="isLocationAllSelected" @change="handleLocationAllChange">
                                                <span style="font-weight: bold;"><%=rb.getString("WeiZhi")%></span>
                                            </el-checkbox>
                                        </div>
                                        <el-checkbox-group class="mairgin-left-30" v-show="columnExpanded.location" v-model="columnForm.location" style="padding-left: 30px;">
                                            <el-checkbox v-for="item in filteredLocationCol" :label="item.code" :key="item.code" :disabled="item.disabled" style="min-width: 140px; margin-bottom: 5px;">{{item.label}}</el-checkbox>
                                        </el-checkbox-group>
                                    </div>
                                    <!-- GPS卫星数 -->
                                    <div>
                                        <div style="padding: 5px 10px; cursor: pointer;" @click="columnExpanded.satellite = !columnExpanded.satellite">
                                            <i :class="{'el-icon-arrow-down': columnExpanded.satellite, 'el-icon-arrow-right': !columnExpanded.satellite}"></i>
                                            <el-checkbox :indeterminate="columnForm.satellite.length > 0 && columnForm.satellite.length < satelliteCol.length" v-model="isSatelliteAllSelected" @change="handleSatelliteAllChange">
                                                <span style="font-weight: bold;"><%=rb.getString("GPSWeiXingShu")%></span>
                                            </el-checkbox>
                                        </div>
                                        <el-checkbox-group class="mairgin-left-30" v-show="columnExpanded.satellite" v-model="columnForm.satellite" style="padding-left: 30px;">
                                            <el-checkbox v-for="item in satelliteCol" :label="item.code" :key="item.code" :disabled="item.disabled" style="min-width: 140px; margin-bottom: 5px;">{{item.label}}</el-checkbox>
                                        </el-checkbox-group>
                                    </div>
                                </div>
                                <!-- 右侧：列排序 -->
                                <div style="flex: 1; padding: 0 0 0 5px; overflow: auto;display: flex; flex-direction: column;">
                                    <div style="font-weight: bold; padding: 10px 0; border-bottom: 1px solid #e9e9e9;"><%=rb.getString("LiePaiXu")%></div>
                                    <draggable v-model="dragColumns" @start="columnConfigDrag=true" @end="columnConfigDrag=false" :animation="200" style="flex: auto; overflow: auto; height: 100%;">
                                        <transition-group type="transition" :name="!columnConfigDrag ? 'flip-list' : null">
                                            <div v-for="col in dragColumns" :key="col.field" 
                                                v-if="selectedColumnCodes.includes(col.field)"
                                                style="padding: 8px 10px; border-bottom: 1px dashed #e9e9e9; cursor: move; display: flex; justify-content: space-between; align-items: center;">
                                                <span>{{col.label}}</span>
                                                <i v-if="!col.disabled" class="el-icon el-icon-close sort-item-op" style="cursor: pointer;zoom: 0.6;" @click="removeColumnFromForm(col.field)"></i>
                                            </div>
                                        </transition-group>
                                    </draggable>
                                </div>
                            </div>
                            <div style="text-align: left; padding: 15px 30px; border-top: 1px solid #e9e9e9;">
                                <el-button size="small" type="primary" @click="applyColumnConfig"><%=rb.getString("QueDing")%></el-button>
                                <el-button size="small" @click="columnConfigVisible = false"><%=rb.getString("QuXiao")%></el-button>
                            </div>
                            <span slot="reference" class="el-icon-operation-settings el-icon" style="position:absolute;left:5px;z-index:99;box-shadow:none;bottom:-27px;cursor:pointer;" @click="openColumnConfig"></span>
                        </el-popover>
                    </div>

                    <div style="height: 100%; flex:auto; overflow: auto;">
                        <!-- 监控列表 -->
                        <vxe-table ref="xTable" 
                            :key="tableKey"
                            id="tableHomeCellList"
                            row-id="small_cell_code"
                            border
                            size="mini"
                            show-overflow="tooltip"
                            show-header-overflow="tooltip"
                            :data="tableData"
                            :height="tableHeight"
                            :loading="tableLoading"
                            :column-config="{resizable: true}"
                            :row-config="{isHover: true, keyField: 'small_cell_code'}"
                            :checkbox-config="{checkField: '_checked', reserve: true, highlight: true}"
                            :scroll-y="{enabled: true, gt: 0, scrollToTopOnChange: false}"
                            :scroll-x="{enabled: true, gt: 0}"
                            :stripe="true"
                            :sort-config="{remote: true, trigger: 'cell', defaultSort: {field: '', order: ''}}"
                            :tooltip-config="{theme: 'light', enterable: true, contentMethod: cellTooltipMethod}"
                            @checkbox-change="handleSelectionChange"
                            @checkbox-all="handleSelectionChange"
                            @sort-change="handleSortChange">
                            <!-- 动态排序的列 -->
                            <template v-for="col in sortedVisibleColumns">
                                <!-- 序号列 -->
                                <vxe-table-column v-if="col.type === 'seq'" :key="col.key" type="seq" title=" " width="50" fixed="left">
                                    <template #default="{ row, rowIndex }">
                                        <span>{{pager.pageSize*(pager.currentPage-1) + 1 + rowIndex}}</span>
                                    </template>
                                </vxe-table-column>
                                <!-- 复选框列 -->
                                <vxe-table-column v-else-if="col.type === 'checkbox'" :key="col.key" type="checkbox" width="40" fixed="left"></vxe-table-column>
                                <!-- 操作列 -->
                                <vxe-table-column v-else-if="col.field === 'enb_operation'" :key="col.key" field="enb_operation" width="70" fixed="left">
                                    <template #default="{ row }">
                                        <div class="el-icon el-icon-circle-setting1" @click="openSettingPage(row)" style="font-size:19px;"></div>
                                        <div v-show="isMonitorWritable" class="el-icon el-icon-operation-more-circle" @click="optClick(row, event)" v-clickoutside="handerClose"></div>
                                    </template>
                                </vxe-table-column>
                                <!-- 连接状态列 -->
                                <vxe-table-column v-else-if="col.field === 'connection_status'" :key="col.key" sortable field="connection_status" width="40" fixed="left">
                                    <template #default="{ row, rowIndex }">
                                        <div v-html="connStatusFmt(row, row['connection_status'], rowIndex)"></div>
                                    </template>
                                </vxe-table-column>
                                <!-- 告警列 -->
                                <vxe-table-column v-else-if="col.field === 'alarm'" :key="col.key" sortable title="<%=rb.getString("GaoJingShu")%>" field="alarm" width="115" fixed="left">
                                    <template #default="{ row }">
                                        <div style="text-align: center;">
                                            <span v-if="row.alarm_serverity == '31001' "  class="alarmCritical alarmListSty" @click="alarmToInfo(row, event)">{{row.alarm_count}}</span>
                                            <span v-else-if="row.alarm_serverity == '31002' "  class="alarmMajor alarmListSty" @click="alarmToInfo(row, event)">{{row.alarm_count}}</span>
                                            <span v-else-if="row.alarm_serverity == '31003' "  class="alarmMinor alarmListSty" @click="alarmToInfo(row, event)">{{row.alarm_count}}</span>
                                            <span v-else-if="row.alarm_serverity == '31004' "  class="alarmWarning alarmListSty" @click="alarmToInfo(row, event)">{{row.alarm_count}}</span>
                                            <span v-else >0</span>
                                        </div>
                                    </template>
                                </vxe-table-column>
                                <!-- serial_number 列 -->
                                <vxe-table-column v-else-if="col.field === 'serial_number'" :key="col.key" sortable title="<%=rb.getString("XiaoZhanBianMa")%>" field="serial_number" width="180" fixed="left"></vxe-table-column>
                                <!-- host_name 列 -->
                                <vxe-table-column v-else-if="col.field === 'host_name'" :key="col.key" sortable title="<%=rb.getString("HostName")%>" field="host_name" width="150" fixed="left">
                                    <template #default="{ row }">
                                        <div v-if="row.device_name_tip === '0'">{{row.host_name}}</div>
                                        <div v-if="row.device_name_tip !== '0'" class="cellNameClass">
                                            <el-popover trigger="click">
                                                <div slot="reference">
                                                    <span class="el-icon el-icon-circle-warning"></span> 
                                                    {{row.host_name}}
                                                </div>
                                                <div style='padding: 30px 20px 20px; position:relative;'>
                                                    <div> <span class="el-icon el-icon-close" @click="closeSyncName" style="top: 10px; position:absolute;"></span>
                                                    <div style='display: flex;'><span style='color: #333; font-size: 14px;'><%=rb.getString("JiZhanCeMingCheng")%> : </span> <span style='font-size: 14px;margin-left: 6px;color: #333;'>{{row.report_host_name}}</span></div>
                                                    <div style='font-size: 12px; color: #333; margin-top: 6px;'><%=rb.getString("ShiFouTongBuMingChengDaoOMC")%><div>
                                                    <div style="margin-top: 10px; text-align: right;">
                                                        <span class="el-button--primary el-button" @click="syncName(row.small_cell_code, row.report_host_name)">
                                                            <%=rb.getString("QueDing")%>
                                                        </span>
                                                        <span class="white el-button" @click="closeSyncName"><%=rb.getString("QuXiao")%></span>
                                                    </div>
                                                </div>
                                            </el-popover>
                                        </div>
                                    </template>
                                </vxe-table-column>
                                <!-- cellId 列 -->
                                <vxe-table-column v-else-if="col.field === 'cellId'" :key="col.key" title="<%=rb.getString("XIAOQUID")%>" field="cellId" width="100">
                                    <template #default="{ row }">
                                        <div v-if="['BM'].includes(row.platformType)">
                                            <div v-if="!['','NULL','null',null,undefined].includes(row.cellId) || !['','NULL','null',null,undefined].includes(row.gsm_cells_cellid)" style="display: flex;align-items: center;">
                                                <el-popover title="All CELL ID" popper-class="mmePopoverClass">
                                                    <span style="color:#4d84ff;" slot="reference">
                                                        [ {{(row.cellId ? row.cellId.split(',').length : 0) + (row.gsm_cells_cellid ? row.gsm_cells_cellid.split(',').length : 0)}}
                                                        <i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;"></i>]
                                                    </span>
                                                    <div v-if="!['','NULL','null',null,undefined].includes(row.cellId)" style="padding:10px 10px 10px;border-top:1px solid #E9E9E9;min-height:30px;display:flex;flex-wrap:wrap;">
                                                        <div v-for="(item,index) in parseCellAndRfStatus(row.cellId)" style="height:30px; font-size: 12px; padding:0 10px; border: 1px solid #E9E9E9; display:flex;align-items: center;margin-left:10px;">
                                                            LTE Cell {{index+1}}: {{item}}
                                                        </div>
                                                    </div>
                                                    <div v-if="!['','NULL','null',null,undefined].includes(row.gsm_cells_cellid)" style="padding:0 10px;min-height:30px;display:flex;flex-wrap:wrap; margin-bottom: 10px;">
                                                        <div v-for="(item,index) in parseCellAndRfStatus(row.gsm_cells_cellid)" style="height:30px; font-size: 12px; padding:0 10px;border: 1px solid #E9E9E9;display:flex;align-items: center;margin-left:10px;">
                                                            GSM Cell {{index+1}}: {{item}}
                                                        </div>
                                                    </div>
                                                </el-popover>
                                            </div>
                                            <div v-else></div>
                                        </div>
                                        <div v-else>{{row.cellId}}</div>
                                    </template>
                                </vxe-table-column>
                                <!-- rf_status 射频开关状态列 -->
                                <vxe-table-column v-else-if="col.field === 'rf_status'" :key="col.key" sortable title="<%=rb.getString("ShePinKaiGuanZhuangTai")%>" field="rf_status" width="150">
                                    <template #default="{ row }">
                                        <div v-if="['BM'].includes(row.platformType)" style="display: flex;align-items: center;">
                                            <div v-if="['','NULL','null',null,undefined].includes(row.rf_status) && ['','NULL','null',null,undefined].includes(row.gsm_cells_rf_state)"></div>
                                            <div v-else-if="row.rf_status == '--' && row.gsm_cells_rf_state == '--'">--</div>
                                            <div v-else-if="(!['','NULL','null',null,undefined].includes(row.rf_status) && row.rf_status != '--') || (!['','NULL','null',null,undefined].includes(row.gsm_cells_rf_state) && row.gsm_cells_rf_state != '--')">
                                                <div v-if="parseCellAndRfStatus(row.rf_status).length > 1 || parseCellAndRfStatus(row.gsm_cells_rf_state).length > 1" style="display: flex;">
                                                    <span v-if="judgeActiveStatusFat(row.rf_status) == '1' && judgeActiveStatusFat(row.gsm_cells_rf_state) == '1'" style='margin-right: 5px;'><%=rb.getString("Kai")%></span>
                                                    <span v-else-if="judgeActiveStatusFat(row.rf_status) == '3' && judgeActiveStatusFat(row.gsm_cells_rf_state) == '3'" class='offStatusCls' style='margin-right: 5px;'><%=rb.getString("Guan")%></span>
                                                    <span v-else class='offOrOnStatusCls' style='margin-right: 5px;'><%=rb.getString("Kai")%></span>
                                                    <el-popover title="All RF Status" popper-class="cellActivePopoverClass" trigger="click">
                                                        <span style="color:#4d84ff;" slot="reference" v-if="parseCellAndRfStatus(row.rf_status).length>1 || parseCellAndRfStatus(row.gsm_cells_rf_state).length>1">
                                                            [ {{judgeActiveNumOrAllNumFat(row.rf_status,'active') + judgeActiveNumOrAllNumFat(row.gsm_cells_rf_state,'active')}}/{{judgeActiveNumOrAllNumFat(row.rf_status,'all') + judgeActiveNumOrAllNumFat(row.gsm_cells_rf_state,'all')}} ]
                                                        </span>
                                                        <div style="padding:10px 10px 0;border-top:1px solid #E9E9E9;min-height:30px;display:flex;flex-wrap:wrap;">
                                                            <div v-for="(item,index) in parseCellAndRfStatus(row.rf_status)" style="height:40px; font-size: 12px;display:flex;align-items: center;margin-left:10px;">
                                                                LTE Cell {{index+1}}:
                                                                <span v-if="item == 'on' || item == '1'" class="onStatusBoxCls" style='margin-left: 5px;'><%=rb.getString("Kai")%></span>
                                                                <span v-if="item == 'off' || item == '0'" class="offStatusBoxCls" style='margin-left: 5px;'><%=rb.getString("Guan")%></span>
                                                            </div>
                                                        </div>
                                                        <div style="padding:0 10px;min-height:30px;display:flex;flex-wrap:wrap;">
                                                            <div v-for="(item,index) in parseCellAndRfStatus(row.gsm_cells_rf_state)" style="height:40px;font-size: 12px;display:flex;align-items: center;margin-left:10px;">
                                                                GSM Cell {{index+1}}:
                                                                <span v-if="item == 'on' || item == '1'" class="onStatusBoxCls" style='margin-left: 5px;'><%=rb.getString("Kai")%></span>
                                                                <span v-if="item == 'off' || item == '0'" class="offStatusBoxCls" style='margin-left: 5px;'><%=rb.getString("Guan")%></span>
                                                            </div>
                                                        </div>
                                                    </el-popover>
                                                </div>
                                            </div>
                                            <div v-else></div>
                                        </div>
                                        <div v-else style="display: flex;align-items: center;">
                                            <div v-if="['','NULL','null',null,undefined].includes(row.rf_status)"></div>
                                            <div v-else-if="row.rf_status == '--'">--</div>
                                            <div v-else-if="!['','NULL','null',null,undefined].includes(row.rf_status) && row.rf_status != '--'">
                                                <div v-if="row.rf_status == 'on' && parseCellAndRfStatus(row.rf_status).length == 1" class='iconFlexCls'>
                                                    <span style='margin-right: 5px;'><%=rb.getString("Kai")%></span>
                                                </div>
                                                <div v-else-if="row.rf_status == 'off' && parseCellAndRfStatus(row.rf_status).length == 1" class='iconFlexCls'>
                                                    <span class='offStatusCls' style='margin-right: 5px;'><%=rb.getString("Guan")%></span>
                                                </div>
                                                <div v-else-if="parseCellAndRfStatus(row.rf_status).length > 1" style="display: flex;">
                                                    <span v-if="judgeActiveStatusFat(row.rf_status) == '1'" style='margin-right: 5px;'><%=rb.getString("Kai")%></span>
                                                    <span v-else-if="judgeActiveStatusFat(row.rf_status) == '2'" class='offOrOnStatusCls' style='margin-right: 5px;'><%=rb.getString("Kai")%></span>
                                                    <span v-else-if="judgeActiveStatusFat(row.rf_status) == '3'" class='offStatusCls' style='margin-right: 5px;'><%=rb.getString("Guan")%></span>
                                                    <el-popover title="<%= rb.getString("DuoXiaoQuZhuangTai")%>" popper-class="cellActivePopoverClass" trigger="click" width='300'>
                                                        <span style="color:#4d84ff;" slot="reference" v-if="parseCellAndRfStatus(row.rf_status).length>1">
                                                            [ {{judgeActiveNumOrAllNumFat(row.rf_status,'active')}}/{{judgeActiveNumOrAllNumFat(row.rf_status,'all')}} ]
                                                        </span>
                                                        <div style="padding:10px;border-top:1px solid #E9E9E9;min-height:100px;display:flex;flex-wrap:wrap;">
                                                            <div v-for="(item,index) in parseCellAndRfStatus(row.rf_status)" style="height:40px;display:flex;align-items: center;margin-left:10px;">
                                                                Cell {{index+1}}:
                                                                <span v-if="item == 'on'" class="onStatusBoxCls" style='margin-left: 5px;'><%=rb.getString("Kai")%></span>
                                                                <span v-if="item == 'off'" class="offStatusBoxCls" style='margin-left: 5px;'><%=rb.getString("Guan")%></span>
                                                            </div>
                                                        </div>
                                                    </el-popover>
                                                </div>
                                            </div>
                                        </div>
                                    </template>
                                </vxe-table-column>
                                <!-- op_state 是否激活列 -->
                                <vxe-table-column v-else-if="col.field === 'op_state'" :key="col.key" sortable title="<%=rb.getString("ShiFouJiHuo")%>" field="op_state" width="160">
                                    <template #default="{ row }">
                                        <div style="display: flex;align-items: center;">
                                            <div v-if="row.product != 'PM-B4860'">
                                                <div v-if="row.op_state == '1' && !['Intel_CR_CA','Intel_CR_TC','MLN_CA', 'BM'].includes(row.platformType)" class='iconFlexCls'>
                                                    <span style='margin-right: 5px;'><%= rb.getString("JiHuo")%></span>
                                                </div>
                                                <div v-if="row.op_state == '0' && !['Intel_CR_CA','Intel_CR_TC','MLN_CA', 'BM'].includes(row.platformType)" class='iconFlexCls'>
                                                    <span class='offStatusCls' style='margin-right: 5px;'><%= rb.getString("QuJiHuo")%></span>
                                                </div>
                                                <div v-if="['Intel_CR_CA','Intel_CR_TC','MLN_CA'].includes(row.platformType)" style="display: flex;">
                                                    <span v-if="judgeActiveStatusFat(row.op_state) == '1'" style='margin-right: 5px;'><%= rb.getString("JiHuo")%></span>
                                                    <span v-if="judgeActiveStatusFat(row.op_state) == '2'" class='offOrOnStatusCls' style='margin-right: 5px;'><%= rb.getString("JiHuo")%></span>
                                                    <span v-if="judgeActiveStatusFat(row.op_state) == '3'" class='offStatusCls' style='margin-right: 5px;'><%= rb.getString("QuJiHuo")%></span>
                                                    <el-popover title="<%= rb.getString("DuoXiaoQuZhuangTai")%>" popper-class="cellActivePopoverClass" trigger="click" width='300'>
                                                        <span style="color:#4d84ff;" slot="reference">
                                                            [ {{judgeActiveNumOrAllNumFat(row.op_state,'active')}}/{{judgeActiveNumOrAllNumFat(row.op_state,'all')}} ]
                                                        </span>
                                                        <div style="padding:10px;border-top:1px solid #E9E9E9;min-height:100px;display:flex;flex-wrap:wrap;">
                                                            <div v-for="(item,index) in parseCellAndRfStatus(row.op_state)" style="height:40px;display:flex;align-items: center;margin-left:10px;">
                                                                Cell {{index+1}}:
                                                                <span v-if="item == '1'" class="onStatusBoxCls" style='margin-left: 5px;'><%= rb.getString("JiHuo")%></span>
                                                                <span v-if="item == '0'" class="offStatusBoxCls" style='margin-left: 5px;'><%= rb.getString("QuJiHuo")%></span>
                                                            </div>
                                                        </div>
                                                    </el-popover>
                                                </div>
                                                <div v-if="['BM'].includes(row.platformType)" style="display: flex;">
                                                    <div>
                                                        <span v-if="['op_state', 'gsm_cells_op_state'].filter(key => !['','NULL','null',null,undefined].includes(row[key])).every(key => judgeActiveStatusFat(row[key]) == '1')" style='margin-right: 5px;'><%= rb.getString("JiHuo")%></span>
                                                        <span v-else-if="['op_state', 'gsm_cells_op_state'].filter(key => !['','NULL','null',null,undefined].includes(row[key])).every(key => judgeActiveStatusFat(row[key]) == '3')" class='offStatusCls' style='margin-right: 5px;'><%= rb.getString("QuJiHuo")%></span>
                                                        <span v-else class='offOrOnStatusCls' style='margin-right: 5px;'><%= rb.getString("JiHuo")%></span>
                                                        <el-popover title="<%= rb.getString("DuoXiaoQuZhuangTai")%>" popper-class="cellActivePopoverClass" trigger="click">
                                                            <span style="color:#4d84ff;" slot="reference">
                                                                [{{['op_state', 'gsm_cells_op_state'].filter(key => !['','NULL','null',null,undefined].includes(row[key])).reduce((sum, key) => sum + judgeActiveNumOrAllNumFat(row[key],'active'), 0)}}/{{['op_state', 'gsm_cells_op_state'].filter(key => !['','NULL','null',null,undefined].includes(row[key])).reduce((sum, key) => sum + judgeActiveNumOrAllNumFat(row[key],'all'), 0)}} ]
                                                            </span>
                                                            <div v-if="!['','NULL','null',null,undefined].includes(row.op_state)" style="padding:10px 10px 0;border-top:1px solid #E9E9E9;min-height:30px;display:flex;flex-wrap:wrap;">
                                                                <div v-for="(item,index) in parseCellAndRfStatus(row.op_state)" style="height:40px;display:flex;align-items: center;margin-left:10px;font-size: 12px;">
                                                                    LTE Cell {{index+1}}:
                                                                    <span v-if="['1', 1].includes(item)" class="onStatusBoxCls" style='margin-left: 5px;'><%= rb.getString("JiHuo")%></span>
                                                                    <span v-if="['0', 0].includes(item)" class="offStatusBoxCls" style='margin-left: 5px;'><%= rb.getString("QuJiHuo")%></span>
                                                                </div>
                                                            </div>
                                                            <div v-if="!['','NULL','null',null,undefined].includes(row.gsm_cells_op_state)" style="padding:0 10px;min-height:30px;display:flex;flex-wrap:wrap;">
                                                                <div v-for="(item,index) in parseCellAndRfStatus(row.gsm_cells_op_state)" style="height:40px;font-size: 12px;display:flex;align-items: center;margin-left:10px;">
                                                                    GSM Cell {{index+1}}:
                                                                    <span v-if="item == '1'" class="onStatusBoxCls" style='margin-left: 5px;'><%= rb.getString("JiHuo")%></span>
                                                                    <span v-if="item == '0'" class="offStatusBoxCls" style='margin-left: 5px;'><%= rb.getString("QuJiHuo")%></span>
                                                                </div>
                                                            </div>
                                                        </el-popover>
                                                    </div>
                                                </div>
                                            </div>
                                            <div v-if="row.product == 'PM-B4860'">
                                                <div class='iconFlexCls'>
                                                    <span v-if="row.op_state == '0'" class='offStatusCls' style='margin-right: 5px;'><%= rb.getString("QuJiHuo")%></span>
                                                    <span v-if="row.op_state != '0'" class='offOrOnStatusCls' style='margin-right: 5px;'><%= rb.getString("JiHuo")%></span>
                                                    <el-popover title="<%= rb.getString("DuoXiaoQuZhuangTai")%>" popper-class="cellActivePopoverClass" trigger="click" width='640' @show="cellActivePopoverShow(row)" @hide="cellActivePopoverHide">
                                                        <span style="color:#4d84ff;" slot="reference">
                                                            [ {{row.op_state}}/{{row.op_state_cell2}} ]
                                                        </span>
                                                        <div style="padding:10px;border-top:1px solid #E9E9E9;min-height:120px;" v-loading="cellActivePopoverLoading">
                                                            <div v-for="item in activeCellsDataList" v-show='item.cellItem.length>0' style="height:40px;display:flex;align-items: center;margin-bottom:10px;">
                                                                <div style="width:110px;">{{item.label}}（<%= rb.getString("BanKaID")%>）</div>
                                                                <div v-for="(items,index) in item.cellItem" style="padding-left:10px;height:40px;width:160px;display:flex;align-items: center;justify-content: center;border:1px solid #E9E9E9;">
                                                                    <div v-if="items.activeStatus == '1'">
                                                                        <span style="margin-right: 5px;">{{items.cellIndex}}</span>
                                                                        <span class="onStatusBoxCls"><%= rb.getString("JiHuo")%></span>
                                                                    </div>
                                                                    <div v-if="items.activeStatus == '0'">
                                                                        <span style="margin-right: 5px;">{{items.cellIndex}}</span>
                                                                        <span class="offStatusBoxCls"><%= rb.getString("QuJiHuo")%></span>
                                                                    </div>
                                                                </div>
                                                            </div>
                                                        </div>
                                                    </el-popover>
                                                </div>
                                            </div>
                                            <el-tooltip v-if="row.delay_avaliable == '0' || row.delay_avaliable == '1'" content="<%=rb.getString("licenseGuoQi")%>">
                                                <span class="warnTips el-icon el-icon-sas-warning" style='margin-left: 5px;'></span>
                                            </el-tooltip>
                                        </div>
                                    </template>
                                </vxe-table-column>
                                <!-- mme_status MME状态列 -->
                                <vxe-table-column v-else-if="col.field === 'mme_status'" :key="col.key" title="<%=rb.getString("MMEZhuangTai")%>" field="mme_status" width="140">
                                    <template #default="{ row }">
                                        <div v-if="['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN','MLN_CA','MLN_SC','MLN_DC'].includes(row.platformType)">
                                            <div v-if="['','NULL','null',null,undefined].includes(row.NEW_MME_STATUS)" style="display: flex;align-items: center;">
                                                <div v-if="['','NULL','null',null,undefined].includes(row.temp_new_mme_status)">
                                                    <span v-html="oldMmeStatusFmt(row, row.mme_status, 'left')"></span>
                                                    <el-popover title="All MME" popper-class="mmePopoverClass" v-if="oldMmeStatusFmt(row, row.mme_status, 'list').length > 0">
                                                        <span style="color:#4d84ff;cursor:pointer;" slot="reference" v-if="oldMmeStatusFmt(row, row.mme_status, 'list').length > 0">
                                                            [ <span v-html="oldMmeStatusFmt(row, row.mme_status, 'right')"></span> ]
                                                        </span>
                                                        <div class="mme-list">
                                                            <i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
                                                            <div class="mme-list-item" v-for="item in oldMmeStatusFmt(row, row.mme_status, 'list')">
                                                                <span v-if="item.status == '1'" style="margin-top:10px;" class="el-icon el-icon-status-MME greenIcon"></span>
                                                                <span v-if="item.status == '0'" style="margin-top:10px;" class="el-icon el-icon-status-MME redIcon"></span>
                                                                <div class="mme-info">
                                                                    <span>MME IP : {{item.mmeIp}}</span>
                                                                    <span v-if="item.status==='0'"><%=rb.getString("MMEZhuangTai")%> : <%= rb.getString("MMEWeiLianJie")%></span>
                                                                    <span v-if="item.status=='1'"><%=rb.getString("MMEZhuangTai")%> : <%= rb.getString("MMEYiLianJie")%></span>
                                                                    <span v-if="item.status=='2'"><%=rb.getString("MMEZhuangTai")%> : --</span>
                                                                    <span v-if="item.status==='' || item.status===null"><%=rb.getString("MMEZhuangTai")%> : </span>
                                                                </div>
                                                            </div>
                                                        </div>
                                                    </el-popover>
                                                </div>
                                                <div v-if="!['','NULL','null',null,undefined].includes(row.temp_new_mme_status)">--</div>
                                            </div>
                                            <div v-if="!['','NULL','null',null,undefined].includes(row.NEW_MME_STATUS)" style="display: flex;align-items: center;">
                                                <span v-html="mmesStatusFmt(row.NEW_MME_STATUS)"></span>
                                                <el-popover title="All MME" popper-class="mmePopoverClass">
                                                    <span style="color:#4d84ff;cursor:pointer;" slot="reference">
                                                        [ {{parseMMEOnNum(row.NEW_MME_STATUS)}}/{{parseMME(row.NEW_MME_STATUS).length}} ]
                                                    </span>
                                                    <div class="mme-list">
                                                        <i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
                                                        <div class="mme-list-item" v-for="item in parseMME(row.NEW_MME_STATUS)">
                                                            <span v-if="item.status == '1'" class="el-icon el-icon-status-MME greenIcon"></span>
                                                            <span v-if="item.status == '0'" class="el-icon el-icon-status-MME redIcon"></span>
                                                            <div class="mme-info">
                                                                <span>MME IP : {{item.mmeIp}}</span>
                                                                <span v-if="item.status==='0'"><%=rb.getString("MMEZhuangTai")%> : <%= rb.getString("MMEWeiLianJie")%></span>
                                                                <span v-if="item.status=='1'"><%=rb.getString("MMEZhuangTai")%> : <%= rb.getString("MMEYiLianJie")%></span>
                                                                <span v-if="item.status=='2'"><%=rb.getString("MMEZhuangTai")%> : --</span>
                                                                <span v-if="item.status==='' || item.status===null"><%=rb.getString("MMEZhuangTai")%> : </span>
                                                                <span><%=rb.getString("PLMN")%> : {{item.plmnId}}</span>
                                                            </div>
                                                        </div>
                                                    </div>
                                                </el-popover>
                                            </div>
                                        </div>
                                        <div v-else>
                                            <div v-if="['','NULL','null',null,undefined].includes(row.NEW_MME_STATUS)" style="display: flex;align-items: center;">
                                                <span v-html="oldMmeStatusFmt(row, row.mme_status, 'left')"></span>
                                                <el-popover title="All MME" popper-class="mmePopoverClass" v-if="oldMmeStatusFmt(row, row.mme_status, 'list').length > 0">
                                                    <span style="color:#4d84ff;cursor:pointer;" slot="reference" v-if="oldMmeStatusFmt(row, row.mme_status, 'list').length > 0">
                                                        [ <span v-html="oldMmeStatusFmt(row, row.mme_status, 'right')"></span> ]
                                                    </span>
                                                    <div class="mme-list">
                                                        <i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
                                                        <div class="mme-list-item" v-for="item in oldMmeStatusFmt(row, row.mme_status, 'list')">
                                                            <span v-if="item.status == '1'" style="margin-top:10px;" class="el-icon el-icon-status-MME greenIcon"></span>
                                                            <span v-if="item.status == '0'" style="margin-top:10px;" class="el-icon el-icon-status-MME redIcon"></span>
                                                            <div class="mme-info">
                                                                <span>MME IP : {{item.mmeIp}}</span>
                                                                <span v-if="item.status==='0'"><%=rb.getString("MMEZhuangTai")%> : <%= rb.getString("MMEWeiLianJie")%></span>
                                                                <span v-if="item.status=='1'"><%=rb.getString("MMEZhuangTai")%> : <%= rb.getString("MMEYiLianJie")%></span>
                                                                <span v-if="item.status=='2'"><%=rb.getString("MMEZhuangTai")%> : --</span>
                                                                <span v-if="item.status==='' || item.status===null"><%=rb.getString("MMEZhuangTai")%> : </span>
                                                            </div>
                                                        </div>
                                                    </div>
                                                </el-popover>
                                            </div>
                                            <div v-if="!['','NULL','null',null,undefined].includes(row.NEW_MME_STATUS)" style="display: flex;align-items: center;">
                                                <span v-html="mmesStatusFmt(row.NEW_MME_STATUS)"></span>
                                                <el-popover title="All MME" popper-class="mmePopoverClass">
                                                    <span style="color:#4d84ff;cursor:pointer;" slot="reference">
                                                        [ {{parseMMEOnNum(row.NEW_MME_STATUS)}}/{{parseMME(row.NEW_MME_STATUS).length}} ]
                                                    </span>
                                                    <div class="mme-list" v-if="!['QAFA','QATA','QAFB'].includes(row.product)">
                                                        <i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
                                                        <div class="mme-list-item" v-for="item in parseMME(row.NEW_MME_STATUS)">
                                                            <span v-if="item.status == '1'" class="el-icon el-icon-status-MME greenIcon"></span>
                                                            <span v-if="item.status == '0'" class="el-icon el-icon-status-MME redIcon"></span>
                                                            <div class="mme-info">
                                                                <span>MME IP : {{item.mmeIp}}</span>
                                                                <span v-if="item.status==='0'"><%=rb.getString("MMEZhuangTai")%> : <%= rb.getString("MMEWeiLianJie")%></span>
                                                                <span v-if="item.status=='1'"><%=rb.getString("MMEZhuangTai")%> : <%= rb.getString("MMEYiLianJie")%></span>
                                                                <span v-if="item.status=='2'"><%=rb.getString("MMEZhuangTai")%> : --</span>
                                                                <span v-if="item.status==='' || item.status===null"><%=rb.getString("MMEZhuangTai")%> : </span>
                                                                <span><%=rb.getString("PLMN")%> : {{item.plmnId}}</span>
                                                            </div>
                                                        </div>
                                                    </div>
                                                    <div class="mme-list" v-if="['QAFA','QATA','QAFB'].includes(row.product)">
                                                        <i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
                                                        <div class="mme-list-item" v-for="item in parseMME(row.NEW_MME_STATUS)">
                                                            <span v-if="item.status == '1'" class="el-icon el-icon-status-MME greenIcon"></span>
                                                            <span v-if="item.status == '0'" class="el-icon el-icon-status-MME redIcon"></span>
                                                            <div class="mme-info">
                                                                <span>MME IP : {{item.mmeIp}}</span>
                                                                <span v-if="item.status==='0'"><%=rb.getString("MMEZhuangTai")%> : <%= rb.getString("MMEWeiLianJie")%></span>
                                                                <span v-if="item.status=='1'"><%=rb.getString("MMEZhuangTai")%> : <%= rb.getString("MMEYiLianJie")%></span>
                                                                <span v-if="item.status=='2'"><%=rb.getString("MMEZhuangTai")%> : --</span>
                                                                <span v-if="item.status==='' || item.status===null"><%=rb.getString("MMEZhuangTai")%> : </span>
                                                                <span>MME Port : {{item.mmePort}}</span>
                                                            </div>
                                                        </div>
                                                    </div>
                                                </el-popover>
                                            </div>
                                        </div>
                                    </template>
                                </vxe-table-column>
                                <!-- plmnid PLMN列 -->
                                <vxe-table-column v-else-if="col.field === 'plmnid'" :key="col.key" title="<%=rb.getString("PLMN")%>" field="plmnid" width="80">
                                    <template #default="{ row, rowIndex }">
                                        <div v-if="['Intel_CR_CA','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN','MLN_CA','MLN_SC','MLN_DC'].includes(row.platformType)">
                                            <div v-if="!['','NULL','null',null,undefined].includes(row.plmnid)" style="display: flex;align-items: center;">
                                                <div v-if="plmnIsJson(row.plmnid) == true">
                                                    <el-popover title="All PLMN" popper-class="mmePopoverClass">
                                                        <span style="color:#4d84ff;" slot="reference">
                                                            [ <span v-html="parsePLMN(row, row.plmnid, rowIndex)"></span>
                                                            <i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;"></i>]
                                                        </span>
                                                        <div class="mme-list specialPlmn" v-html="plmnNewFmt(row, row.plmnid, rowIndex)">
                                                            <i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
                                                        </div>
                                                    </el-popover>
                                                </div>
                                                <div v-else>
                                                    <div v-html="parseStrPLMN(row, row.plmnid, rowIndex)"></div>
                                                </div>
                                            </div>
                                            <div v-else>
                                                <div v-html="plmnSpecialFmt(row, row.plmnid, rowIndex)" style="display: flex;white-space: nowrap;"></div>
                                            </div>
                                        </div>
                                        <div v-else-if="['BM'].includes(row.platformType)">
                                            <div v-if="!['','NULL','null',null,undefined].includes(row.plmnid) || !['','NULL','null',null,undefined].includes(row.gsm_cells_plmnid)" style="display: flex;align-items: center;">
                                                <el-popover title="All PLMN" popper-class="mmePopoverClass">
                                                    <span style="color:#4d84ff;" slot="reference">
                                                        [ {{(row.plmnid && plmnIsJson(row.plmnid) == true ? parsePLMN(row, row.plmnid, rowIndex) : (row.plmnid ? row.plmnid.split(',').length : 0)) + (row.gsm_cells_plmnid ? row.gsm_cells_plmnid.split(',').length : 0)}}
                                                        <i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;"></i>]
                                                    </span>
                                                    <div v-if="!['','NULL','null',null,undefined].includes(row.plmnid)" style="padding:10px 10px 10px;border-top:1px solid #E9E9E9;min-height:30px;flex-wrap:wrap;">
                                                        <div v-if="plmnIsJson(row.plmnid) == true" v-html="lteOrGsmplmnNewFmt(row, row.plmnid, rowIndex)" style="display:flex;">
                                                            <i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
                                                        </div>
                                                        <div v-else style="display:flex;flex-wrap:wrap;">
                                                            <div v-for="(item,index) in parseCellAndRfStatus(row.plmnid)" style="height:30px; font-size: 12px; padding:0 10px; border: 1px solid #E9E9E9; display:flex;align-items: center;margin-left:10px;">
                                                                LTE Cell {{index+1}}: {{item}}
                                                            </div>
                                                        </div>
                                                    </div>
                                                    <div v-if="!['','NULL','null',null,undefined].includes(row.gsm_cells_plmnid)" style="padding:0 10px;min-height:30px;display:flex;flex-wrap:wrap; margin-bottom: 10px;">
                                                        <div v-for="(item,index) in parseCellAndRfStatus(row.gsm_cells_plmnid)" style="height:30px; font-size: 12px; padding:0 10px;border: 1px solid #E9E9E9;display:flex;align-items: center;margin-left:10px;">
                                                            GSM Cell {{index+1}}: {{item}}
                                                        </div>
                                                    </div>
                                                </el-popover>
                                            </div>
                                            <div v-else></div>
                                        </div>
                                        <div v-else>
                                            <div v-html="plmnFmt(row, row.plmnid, rowIndex)"></div>
                                        </div>
                                    </template>
                                </vxe-table-column>
                                <!-- ue_count UE数列 -->
                                <vxe-table-column v-else-if="col.field === 'ue_count'" :key="col.key" sortable title="<%=rb.getString("UEShu")%>" field="ue_count" width="80">
                                    <template #default="{ row, rowIndex }">
                                        <div v-if="['BM'].includes(row.platformType)">
                                            <div v-if="!['','NULL','null',null,undefined].includes(row.ue_count) || !['','NULL','null',null,undefined].includes(row.gsm_cells_ue)">
                                                <div v-if="row.ue_count == -1 && row.gsm_cells_ue == -1">--</div>
                                                <div v-else style="display: flex;align-items: center;">
                                                    <el-popover title="" popper-class="mmePopoverClass">
                                                        <span v-if="(Number(row.ue_count == -1 ? 0 : row.ue_count) + Number(row.gsm_cells_ue == -1 ? 0 : row.gsm_cells_ue)) == 0" slot="reference">0</span>
                                                        <span v-else-if="(Number(row.ue_count == -1 ? 0 : row.ue_count) + Number(row.gsm_cells_ue == -1 ? 0 : row.gsm_cells_ue)) > 0" style="color:#4d84ff;cursor:pointer;" slot="reference">
                                                            {{Number(row.ue_count == -1 ? 0 : row.ue_count) + Number(row.gsm_cells_ue == -1 ? 0 : row.gsm_cells_ue)}}
                                                        </span>
                                                        <span v-else slot="reference">--</span>
                                                        <div v-if="!['','NULL','null',null,undefined,-1].includes(row.ue_count) && row.ue_count != -1">
                                                            <div v-if="Number(row.ue_count) > 0" style="padding:10px 10px 0;border-top:1px solid #E9E9E9;min-height:30px;display:flex;flex-wrap:wrap;">
                                                                LTE Count: <div style="color:#4d84ff;" v-html="ueCountFmt(row, row.ue_count, rowIndex)"></div>
                                                            </div>
                                                            <div v-else-if="Number(row.ue_count) == 0" style="padding:10px 10px 0;border-top:1px solid #E9E9E9;min-height:30px;display:flex;flex-wrap:wrap;">
                                                                LTE Count: 0
                                                            </div>
                                                        </div>
                                                        <div v-if="!['','NULL','null',null,undefined,-1].includes(row.gsm_cells_ue) && row.gsm_cells_ue != -1" style="padding:0 10px;min-height:30px;display:flex;flex-wrap:wrap; margin-top:8px">
                                                            GSM Count: {{row.gsm_cells_ue}}
                                                        </div>
                                                    </el-popover>
                                                </div>
                                            </div>
                                            <div v-else>--</div>
                                        </div>
                                        <div v-else>
                                            <div v-html="ueCountFmt(row, row.ue_count, rowIndex)"></div>
                                        </div>
                                    </template>
                                </vxe-table-column>
                                <!-- euCountStr EU数列 -->
                                <vxe-table-column v-else-if="col.field === 'euCountStr'" :key="col.key" title="<%=rb.getString("EUShu")%>" field="euCountStr" width="80">
                                    <template #default="{ row, rowIndex }">
                                        <div v-html="euruCountFmt(row, row.euCountStr, rowIndex)"></div>
                                    </template>
                                </vxe-table-column>
                                <!-- ruCountStr RU数列 -->
                                <vxe-table-column v-else-if="col.field === 'ruCountStr'" :key="col.key" title="<%=rb.getString("RUShu")%>" field="ruCountStr" width="80">
                                    <template #default="{ row, rowIndex }">
                                        <div v-html="euruCountFmt(row, row.ruCountStr, rowIndex)"></div>
                                    </template>
                                </vxe-table-column>
                                <!-- cpe_connect CPE连接数列 -->
                                <vxe-table-column v-else-if="col.field === 'cpe_connect'" :key="col.key" title="<%=rb.getString("CPELianJieShu")%>" field="cpe_connect" width="100">
                                    <template #default="{ row, rowIndex }">
                                        <div v-html="cpeCountFmt(row, row.cpe_connect, rowIndex)"></div>
                                    </template>
                                </vxe-table-column>
                                <!-- cell_ip IP列 -->
                                <vxe-table-column v-else-if="col.field === 'cell_ip'" :key="col.key" sortable title="<%=rb.getString("IPDiZhi")%>" field="cell_ip" width="120">
                                    <template #default="{ row, rowIndex }">
                                        <div v-html="ipAddrFmt(row, row.cell_ip, rowIndex)"></div>
                                    </template>
                                </vxe-table-column>
                                <!-- product 产品类型标识列 -->
                                <vxe-table-column v-else-if="col.field === 'product'" :key="col.key" sortable title="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" field="product" width="120">
                                    <template #default="{ row, rowIndex }">
                                        <div v-html="productFmt(row, row.product, rowIndex)"></div>
                                    </template>
                                </vxe-table-column>
                                <!-- module_type 设备型号名列 -->
                                <vxe-table-column v-else-if="col.field === 'module_type'" :key="col.key" sortable title="<%=rb.getString("SheBeiXingHaoMing")%>" field="module_type" width="120">
                                    <template #default="{ row, rowIndex }">
                                        <div v-html="capablityFmt(row, row.module_type, rowIndex)"></div>
                                    </template>
                                </vxe-table-column>
                                <!-- site_id 站点ID列 -->
                                <vxe-table-column v-else-if="col.field === 'site_id'" :key="col.key" sortable :title="siteIdLabel" field="site_id" width="80"></vxe-table-column>
                                <!-- sub_station_name 站点名称列 -->
                                <vxe-table-column v-else-if="col.field === 'sub_station_name'" :key="col.key" :title="siteNameLabel" field="sub_station_name" width="120"></vxe-table-column>
                                <!-- EARFCNDLINUSE 频点列 -->
                                <vxe-table-column v-else-if="col.field === 'EARFCNDLINUSE'" :key="col.key" sortable title="<%=rb.getString("PinDian")%>" field="EARFCNDLINUSE" width="130">
                                    <template #default="{ row, rowIndex }">
                                        <div v-html="earfcnFmt(row, row.EARFCNDLINUSE, rowIndex)"></div>
                                    </template>
                                </vxe-table-column>
                                <!-- synStatus 同步状态列 -->
                                <vxe-table-column v-else-if="col.field === 'synStatus'" :key="col.key" sortable title="<%=rb.getString("TongBuZhuangTai")%>" field="synStatus" width="160">
                                    <template #default="{ row, rowIndex }">
                                        <div v-html="syncStatusFmt(row, row.synStatus, rowIndex)"></div>
                                    </template>
                                </vxe-table-column>
                                <!-- pm_report_status KPI上报状态列 -->
                                <vxe-table-column v-else-if="col.field === 'pm_report_status'" :key="col.key" sortable title="<%=rb.getString("KPIShangBaoZhuangTai")%>" field="pm_report_status" width="135">
                                    <template #default="{ row, rowIndex }">
                                        <div v-html="kpiStatusFmt(row, row.pm_report_status, rowIndex)" style="display: flex; align-items: center;"></div>
                                    </template>
                                </vxe-table-column>
                                <!-- gps_satellite_count GPS卫星数列 -->
                                <vxe-table-column v-else-if="col.field === 'gps_satellite_count'" :key="col.key" sortable title="<%=rb.getString("GPSWeiXingShu")%>" field="gps_satellite_count" width="85">
                                    <template #default="{ row }">
                                        <div v-if='row.hasSatelliteDetail == "true"'>
                                            <a style='color:#1DA3FC;text-decoration:underline' href='#' @click='getEnbGPSSignalData(row)'>{{row.gps_satellite_count}}</a>
                                        </div>
                                        <div v-if='row.hasSatelliteDetail != "true"'>{{row.gps_satellite_count}}</div>
                                    </template>
                                </vxe-table-column>
                                <!-- remark 备注列 -->
                                <vxe-table-column v-else-if="col.field === 'remark'" :key="col.key" field="remark" width="185">
                                    <template #header>
                                        <div class="remarkHeaderCls">
                                            <span v-if="!editingRemarkLabel" style="display: flex; align-items:center;">
                                                <span :title="currentRemarkLabel" style="max-width: 140px; display: inline-block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; vertical-align: middle;">
                                                    {{currentRemarkLabel}}
                                                </span>
                                                <i class="el-icon el-icon-operation-edit" @click="startEditRemarkLabel" style="cursor: pointer; margin-left: 5px; color: #409EFF;"></i>
                                            </span>
                                            <span v-else>
                                                <el-input v-model="remarkLabelInput" size="mini" style="width: 150px;" maxlength="30" @keyup.enter.native="saveRemarkLabel">
                                                    <template slot="suffix">
                                                        <i @click="saveRemarkLabel" class="el-icon el-icon-operation-defaultBeta" style="cursor: pointer; margin-right: 5px;"></i>
                                                        <i @click="cancelEditRemarkLabel" class="el-icon el-icon-deactivate" style="cursor: pointer;"></i>
                                                    </template>
                                                </el-input>
                                            </span>
                                        </div>
                                    </template>
                                    <template #default="{ row }">
                                        {{row.remark}}
                                    </template>
                                </vxe-table-column>
                                <!-- available_rate 小区可用占比列 -->
                                <vxe-table-column v-else-if="col.field === 'available_rate'" :key="col.key" title="<%=rb.getString("XiaoQuKeYongZhanBi")%>" field="available_rate" width="70">
                                    <template #default="{ row }">
                                        <div v-if="row.available_rate=='--'">--</div>
                                        <div v-else>
                                            <a style='color:#1DA3FC;text-decoration:underline' href='#' @click='getUnuseTime(row)'>{{row.available_rate}}</a>
                                        </div>
                                    </template>
                                </vxe-table-column>
                                <!-- halob_flag Halo X列 -->
                                <vxe-table-column v-else-if="col.field === 'halob_flag'" :key="col.key" sortable title="" field="halob_flag" width="100">
                                    <template #header>
                                        Halo <span class="haloXIconBox">X</span>
                                    </template>
                                    <template #default="{ row }">
                                        <div v-if="row.form == '0'|| row.form == '1' || row.form == '2'">
                                            HaloD
                                            <el-popover id="halodCellPopover" title="" popper-class="halodCellPopoverClass" trigger="click" width='240' @show="halodCellPopoverShow(row)" @hide="halodCellPopoverHide">
                                                <span class="haloDModeCodeCls" style="cursor: pointer;" slot="reference">{{row.form}}</span>
                                                <div style="padding:10px;border-top:1px solid #E9E9E9;min-height:120px;" v-loading="halodCellPopoverLoading">
                                                    <div style="display: flex;align-items: center;margin-bottom:10px;">
                                                        <div class="buttonBoxCls">
                                                            <span v-if="halodCellDataList && halodCellDataList.length >0 && halodCellDataList[0].lockStatus == '0'" @click="lockBtnClick('lock',row)" class="el-icon el-icon-operation-unlock"></span>
                                                            <span v-if="halodCellDataList && halodCellDataList.length >0 && halodCellDataList[0].lockStatus == '1'" @click="lockBtnClick('unlock',row)" class="el-icon el-icon-common-lock"></span>
                                                        </div>
                                                        <div class="buttonBoxCls" @click="filterHalodClick"><span class="el-icon el-icon-operation-alarm-filter"></span></div>
                                                    </div>
                                                    <div style="padding:10px;border:1px dashed #E9E9E9;">
                                                        <div v-for="item in halodCellDataList" style="height:30px;display:flex;align-items: center;">
                                                            <span class="itemValueCls">{{item.serialNumber}}</span>
                                                            <span class="haloDModeCodeCls">{{item.form}}</span>
                                                        </div>
                                                    </div>
                                                </div>
                                            </el-popover>
                                        </div>
                                        <div v-else>
                                            <div v-if="row.halob_flag == '1'" style="display: flex;align-items: center;">
                                                HaloB<span class='el-icon el-icon-status-enable' style='margin-left:3px;font-size: 20px;'></span>
                                            </div>
                                            <div v-else-if="row.halob_flag == '0'" style="display: flex;align-items: center;">
                                                HaloB<span class='el-icon el-icon-status-disable' style='margin-left:3px;font-size: 20px;'></span>
                                            </div>
                                            <div v-else>--</div>
                                        </div>
                                    </template>
                                </vxe-table-column>
                                <!-- gps_longitude GPS经度列 -->
                                <vxe-table-column v-else-if="col.field === 'gps_longitude'" :key="col.key" title="<%=rb.getString("GPSJingDu")%>" field="gps_longitude" width="100">
                                    <template #default="{ row, rowIndex }">
                                        <div v-if="row.gps_modify_flag != '1'">{{row.gps_longitude}}</div>
                                        <div v-else>
                                            <div v-if="row.gps_longitude === undefined" :key="rowIndex">
                                                {{row.modify_longitude == undefined? row.gps_longitude : row.modify_longitude}}
                                            </div>
                                            <div v-if="row.gps_longitude !== undefined" class="cellNameClass" :key="rowIndex">
                                                <el-popover trigger="click">
                                                    <div slot="reference">
                                                        <span class="el-icon el-icon-circle-warning"></span>
                                                        {{row.modify_longitude}}
                                                    </div>
                                                    <div class="gpsSyncPopoverContent">
                                                        <span @click="closeSyncName" class="el-icon el-icon-close gpsSyncPopoverClose"></span>
                                                        <div style="margin-bottom: 10px;">
                                                            <%=rb.getString("JingDu")%>: {{row.gps_longitude}}&nbsp;&nbsp;
                                                            <%=rb.getString("WeiDu")%>: {{row.gps_latitude}}&nbsp;&nbsp;
                                                            <%=rb.getString("GaoDu")%>: {{row.gps_height}}
                                                        </div>
                                                        <div style="margin-bottom: 10px;"><%=rb.getString("TongBuGPSTiShi")%></div>
                                                        <div style="text-align: right;margin: 0px;">
                                                            <el-button type="primary" size="mini" @click="synchronizeGPS(row.small_cell_code)">
                                                                <%=rb.getString("QueDing")%>
                                                            </el-button>
                                                            <el-button size="mini" @click="closeSyncName"><%=rb.getString("QuXiao")%></el-button>
                                                        </div>
                                                    </div>
                                                </el-popover>
                                            </div>
                                        </div>
                                    </template>
                                </vxe-table-column>
                                <!-- gps_latitude GPS纬度列 -->
                                <vxe-table-column v-else-if="col.field === 'gps_latitude'" :key="col.key" title="<%=rb.getString("GPSWeiDu")%>" field="gps_latitude" width="90">
                                    <template #default="{ row, rowIndex }">
                                        <div v-if="row.gps_modify_flag != '1'">{{row.gps_latitude}}</div>
                                        <div v-else>
                                            <div v-if="row.gps_latitude === undefined" :key="rowIndex">
                                                {{row.modify_latitude == undefined? row.gps_latitude : row.modify_latitude}}
                                            </div>
                                            <div v-if="row.gps_latitude !== undefined" class="cellNameClass" :key="rowIndex">
                                                <el-popover trigger="click">
                                                    <div slot="reference">
                                                        <span class="el-icon el-icon-circle-warning"></span>
                                                        {{row.modify_latitude}}
                                                    </div>
                                                    <div class="gpsSyncPopoverContent">
                                                        <span @click="closeSyncName" class="el-icon el-icon-close gpsSyncPopoverClose"></span>
                                                        <div style="margin-bottom: 10px;">
                                                            <%=rb.getString("JingDu")%>: {{row.gps_longitude}}&nbsp;&nbsp;
                                                            <%=rb.getString("WeiDu")%>: {{row.gps_latitude}}&nbsp;&nbsp;
                                                            <%=rb.getString("GaoDu")%>: {{row.gps_height}}
                                                        </div>
                                                        <div style="margin-bottom: 20px;"><%=rb.getString("TongBuGPSTiShi")%></div>
                                                        <div style="text-align: right;margin: 0px;">
                                                            <el-button type="primary" size="mini" @click="synchronizeGPS(row.small_cell_code)">
                                                                <%=rb.getString("QueDing")%>
                                                            </el-button>
                                                            <el-button size="mini" @click="closeSyncName"><%=rb.getString("QuXiao")%></el-button>
                                                        </div>
                                                    </div>
                                                </el-popover>
                                            </div>
                                        </div>
                                    </template>
                                </vxe-table-column>
                                <!-- gps_height GPS高度列 -->
                                <vxe-table-column v-else-if="col.field === 'gps_height'" :key="col.key" title="<%=rb.getString("GPSGaoDu")%>" field="gps_height" width="70">
                                    <template #default="{ row, rowIndex }">
                                        <div v-if="row.gps_modify_flag != '1'">{{row.gps_height}}</div>
                                        <div v-else>
                                            <div v-if="row.gps_height === undefined" :key="rowIndex">
                                                {{row.modify_height == undefined? row.gps_height : row.modify_height}}
                                            </div>
                                            <div v-if="row.gps_height !== undefined" class="cellNameClass" :key="rowIndex">
                                                <el-popover trigger="click">
                                                    <div slot="reference">
                                                        <span class="el-icon el-icon-circle-warning"></span>
                                                        {{row.modify_height}}
                                                    </div>
                                                    <div class="gpsSyncPopoverContent">
                                                        <span @click="closeSyncName" class="el-icon el-icon-close gpsSyncPopoverClose"></span>
                                                        <div style="margin-bottom: 10px;">
                                                            <%=rb.getString("JingDu")%>: {{row.gps_longitude}}&nbsp;&nbsp;
                                                            <%=rb.getString("WeiDu")%>: {{row.gps_latitude}}&nbsp;&nbsp;
                                                            <%=rb.getString("GaoDu")%>: {{row.gps_height}}
                                                        </div>
                                                        <div style="margin-bottom: 20px;"><%=rb.getString("TongBuGPSTiShi")%></div>
                                                        <div style="text-align: right;margin: 0px;">
                                                            <el-button type="primary" size="mini" @click="synchronizeGPS(row.small_cell_code)">
                                                                <%=rb.getString("QueDing")%>
                                                            </el-button>
                                                            <el-button size="mini" @click="closeSyncName"><%=rb.getString("QuXiao")%></el-button>
                                                        </div>
                                                    </div>
                                                </el-popover>
                                            </div>
                                        </div>
                                    </template>
                                </vxe-table-column>
                                <!-- validity 有效期列 -->
                                <vxe-table-column v-else-if="col.field === 'validity'" :key="col.key" title="<%=rb.getString("YouXiaoQi")%>" field="validity" width="125">
                                    <template #default="{ row, rowIndex }">
                                        <div v-html="validityFmt(row, row.validity, rowIndex)"></div>
                                    </template>
                                </vxe-table-column>
                                <!-- lock_status 锁定状态列 -->
                                <vxe-table-column v-else-if="col.field === 'lock_status'" :key="col.key" title="<%=rb.getString("SuoDingZhuangTai")%>" field="lock_status" width="90">
                                    <template #default="{ row }">
                                        <div v-if="row.lock_status=='--'">--</div>
                                        <div v-else-if="row.lock_status=='0'">
                                            <div style='display: flex; align-items: center;'><span @click='lockUnlockBtnClick(row)' class='el-icon el-icon-status-unlock' style='margin-right:5px;font-size:18px; cursor: pointer;'></span><%=rb.getString("YouXiaoQiJieSuo")%></div>
                                        </div>
                                        <div v-else>
                                            <div v-if="row.lock_type == '1'" style='display: flex; align-items: center;'>
                                                <el-popover popper-class='infoTipPover' placement="bottom" width="300" trigger="hover">
                                                    <div class="apn-list-item">
                                                        <div class="apn-info">
                                                            <div class='marginRight16'>
                                                                <span class="nameTip"><%=rb.getString("SuoDingYuanYin")%>: </span>
                                                                <span>{{row.lock_reason}}</span>
                                                            </div>
                                                        </div>
                                                    </div>
                                                    <div slot="reference"><span class='el-icon el-icon-operation-lock lockStyle'></span><%=rb.getString("YouXiaoQiSuoDing")%></div>
                                                </el-popover>
                                            </div>
                                            <div v-else>
                                                <div style='display: flex; align-items: center;'><span @click='lockUnlockBtnClick(row)' class='el-icon el-icon-operation-lock lockStyle'></span><%=rb.getString("YouXiaoQiSuoDing")%></div>
                                            </div>
                                        </div>
                                    </template>
                                </vxe-table-column>
                                <!-- 普通列 - 根据 field 动态渲染 -->
                                <vxe-table-column v-else 
                                    :key="col.key"
                                    :field="col.field" 
                                    :title="col.label" 
                                    :width="col.width" 
                                    :sortable="col.sortable">
                                </vxe-table-column>
                            </template>
                        </vxe-table>
                    </div>
                    <!-- 分页组件 -->
                    <el-pagination
                        small
                        :page-size="pager.pageSize"
                        :total="pager.total"
                        :pager-count="3"
                        :page-sizes="[50,100,200]"
                        layout="total, sizes, prev, pager, next, jumper,slot"
                        @size-change="sizeChange"
                        @current-change="currentChage">
                        <span @click="refreshList">
                            <i class="el-icon el-icon-common-refresh" style="margin-left: 10px;cursor: pointer;line-height:22px;font-size:14px;"></i>
                        </span>
                    </el-pagination>
                    
                    <el-cmenu ref="menu" :data="menus" @click="menuClick"></el-cmenu>
                    <div style="position: absolute;bottom: 8px;z-index: 100;display: flex;right: 100px;">
                        <div class="status-statistic active" style="border-right: none;">
                            <span class="el-icon el-icon-status-conn-on"></span>
                            <span style="margin: 0 10px;"><%=rb.getString("ShiFouZaiXian")%></span> 
                            <span id="online_count_rate" style="margin-left: 15px;"></span>
                        </div>
                        <div class="status-statistic active" style="border-right: none;">
                            <span class="el-icon el-icon-status-active"></span>
                            <span style="margin: 0 10px;"><%=rb.getString("JiHuo")%> </span> 
                            <span id="active_count_rate" style="margin-left: 15px;"></span>
                        </div>
                        <div class="status-statistic active" style="border-right: none;">
                            <span class="el-icon el-icon-status-MME"></span>
                            <span style="margin: 0 10px;">MME <%=rb.getString("LianJieZhengChang")%></span> 
                            <span id="mme_count_rate"></span>
                        </div>
                    </div>
                </div>
            </el-tab-pane>
            <el-tab-pane v-if="gsmEnable == 'true'" label='GSM' name="GSM">
                <div id="gsmMonitor" class="flex-item-cls" style="min-width: 1060px;"></div>
            </el-tab-pane>
        </el-tabs>

		<el-slide ref="settingPage" class="settingSlide" 
           	:url="settingUrl" width="80%"
            :footer="false" 
            :header="false">
        </el-slide>
		
        <!-- sliders -->
        <el-slide ref="ueCount" :title="ueslide.title" :height='ueslide.height' :footer="false" @cancel="closeUeSlide" class='commonWarp'>
            <el-ctable :data="ueslide.data" :pagination="false">
                <el-table-column label="UEID" prop="ue_id" width="100"></el-table-column>
                <el-table-column label="IMSI" prop="imsi" width="150"></el-table-column>
                <el-table-column label="VMAC" prop="vmac" width="130"></el-table-column>
                <el-table-column label="<%=rb.getString("CPEName")%>" prop="cpe_name" width="150"></el-table-column>
                <el-table-column label="<%=rb.getString("XiaXingTunTuLv")%>" width="140" prop="downlink_rate"></el-table-column>
                <el-table-column label="<%=rb.getString("ShangXingTunTuLv")%>" width="135" prop="uplink_rate"></el-table-column>
                <el-table-column label="<%=rb.getString("IPDiZhi")%>" prop="ip" width="140"></el-table-column>
                <el-table-column label="<%=rb.getString("DuanKou")%>" prop="port" width="80"></el-table-column>
                <el-table-column label="<%=rb.getString("ShangXingSinr")%>" prop="ulsinr" width="100"></el-table-column>

                <el-table-column v-if="!ueslide.is436q" label="<%=rb.getString("XiaXingCqi")%>" prop="dlcqi" width="100"></el-table-column>

                <el-table-column v-if="ueslide.is436q" label="P_Dlcqi" prop="p_dlcqi" width="100"></el-table-column>
                <el-table-column v-if="ueslide.is436q" label="S_Dlcqi" prop="s_dlcqi" width="100"></el-table-column>

                <el-table-column label="<%=rb.getString("ShangXingmcs")%>" prop="ulmcs" width="100"></el-table-column>

                <el-table-column v-if="!ueslide.is436q" label="<%=rb.getString("XiaXingmcs")%>" prop="dlmcs" width="100"></el-table-column>

                <el-table-column v-if="ueslide.is436q" label="P_Dlmcs" prop="p_dlmcs" width="100"></el-table-column>
                <el-table-column v-if="ueslide.is436q" label="S_Dlmcs" prop="s_dlmcs" width="100"></el-table-column>

                <el-table-column label="<%=rb.getString("FaSongGongLv")%>(dBm)" prop="txpower" width="100"></el-table-column>
                <el-table-column label="<%=rb.getString("ShangXingbler")%>(%)" prop="uplink_bler" width="120"></el-table-column>

                <el-table-column v-if="ueslide.is436q" label="P_TB1_Downlink_BLER(%)" prop="p1_downlink_bler" width="165"></el-table-column>
                <el-table-column v-if="ueslide.is436q" label="P_TB2_Downlink_BLER(%)" prop="p2_downlink_bler" width="165"></el-table-column>
                <el-table-column v-if="ueslide.is436q" label="S_TB1_Downlink_BLER(%)" prop="s1_downlink_bler" width="165"></el-table-column>
                <el-table-column v-if="ueslide.is436q" label="S_TB2_Downlink_BLER(%)" prop="s2_downlink_bler" width="165"></el-table-column>

                <el-table-column v-if="!ueslide.is436q" label="<%=rb.getString("XiaXingbler")%>(%)" prop="downlink_bler" width="165"></el-table-column>

                <el-table-column label="<%=rb.getString("LuJingSunHao")%>(dBm)" prop="pathloss" width="120"></el-table-column>
                <el-table-column v-if='ueS1apId == true' label="UE_S1AP_ID" prop="ue_s1ap_id" width="120"></el-table-column>
				<el-table-column v-if='mmeS1apId == true' label="MME_S1AP_ID" prop="mme_s1ap_id" width="120"></el-table-column>
            </el-ctable>
        </el-slide>

        <el-slide ref="cpeCount" class="no-padding"
            :title="cpeslide.title" 
            :url="cpeslide.url" 
            :footer="false" 
            :header="false">
        </el-slide>

        <el-slide ref="activeRatio" :title="activeslide.title" :footer="false" @cancel="closeActive">
            <div style='display:flex;flex-direction:column;flex:1 1 auto;height:100%;overflow:auto;'>
                <div id="echart_activeRatio" style="min-height: 300px; width: 100%;"></div>
                <div class="list_activeRatio" style="height: 100%;">
                    <el-ctable id="table_activeRatio" :data="activeslide.data" :pagination="false">
                        <el-table-column label='<%=rb.getString("RiQi")%>' prop='days'></el-table-column>
                        <el-table-column label='<%=rb.getString("BuKeYongShiJianDuan")%>' prop='nousedTime'></el-table-column>
                    </el-ctable>
                </div>
            </div>
        </el-slide>

        <el-slide ref="info" class="no-padding no-border slidebarPanel"
            :title="infoslide.title" 
            :url="infoslide.url" 
            :footer="false" 
            :header="false">
        </el-slide>

        <el-slide ref="period" class="no-padding no-border slidebarPanel" 
            :url="periodslide.url" 
            :header="false" 
            :footer="false">
        </el-slide>

        <el-slide ref="distribute" class="no-padding no-border slidebarPanel" 
            :url="distributeslide.url" 
            :header="false" 
            :footer="false">
        </el-slide>
        <el-slide ref="setting" class="no-padding no-border slidebarPanel" 
            :url="enbSettingSlide.url" 
            :header="true" 
            title='<%=rb.getString("SheZhi")%>'
            :footer="true" @cancel="cancelSetting" @ok="saveSetting">
        </el-slide>

        <el-slide ref="satelite" class="no-padding"
            title='<%=rb.getString("GPSWeiXingShu")%>'
            @cancel="closeSatellite"
            :footer="false" >
            <el-ctable :url="satelliteURL" :class="{loading: sateloading}">
                <el-table-column label='<%=rb.getString("WeiXingHao")%>' prop='gnssSvId'></el-table-column>
                <el-table-column label='<%=rb.getString("WeiXingXinHao")%>(dB-Hz)' prop='snr'></el-table-column>
            </el-ctable>
        </el-slide>

        <!--激活/取消激活 弹窗-->
		<el-dialog title="<%=rb.getString("QueRen")%>" id="activeCellDialog" :visible.sync="showActiveCellDialog" top="30vh" ref="activeCellDialog" 
			width="660" :close-on-click-modal="false"  @close='closeActiveCellDialog' append-to-body>
				<div style="margin-bottom:20px;">确认激活/取消激活？</div>
				<div style="border:1px solid #e9e9e9;padding:10px;min-height:100px;" v-loading='activeCellLialogLoading'>
						<div v-for="item in cellInfosList" class="borderCardItemCls" v-show='item.cellItem.length>0'>
							<span class="borderCardItemLabelCls">{{item.label}}（<%= rb.getString("BanKaID")%>）</span>
							<div v-for="(items,index) in item.cellItem"  style="display:inline-block;margin-right:10px;width:150px;font-size:12px;">
								<el-checkbox v-model="items.activeStatus" :true-label="1" :false-label="0">
									{{index+1}}
								</el-checkbox>
								<div v-if="items.oldStatus == '0'" style="display:inline-block;">
									(<span class="el-icon el-icon-status-active redIcon"></span><%= rb.getString("QuJiHuo")%>)
								</div>
								<div v-if="items.oldStatus == '1'" style="display:inline-block;">
									(<span class="el-icon el-icon-status-active greenIcon"></span><%= rb.getString("JiHuo")%>)
								</div>
							</div>
						</div>
				</div>
				<span slot="footer">
					<div>
						<el-button type="primary" @click="activeCellSubmit"><%=rb.getString("QueDing")%></el-button>
						<el-button @click="closeActiveCellDialog"><%=rb.getString("QuXiao")%></el-button>
					</div>
				</span>
		</el-dialog>

        <el-dialog title="<%=rb.getString("QueRen")%>" top="30vh" width="550"
            :visible.sync="collectMessageShow" 
            :modal="false"
            :close-on-click-modal="false">
            <el-form :model="collectForm">
                <div>{{confirmTips}}</div>
                <el-form-item label='<%=rb.getString("ChiXuShiChang")%>' style="display: flex;align-items: center;margin: 5px 0px;">
                    <el-select v-model="collectForm.collectInterval" placeholder="Select time" class="collect-select">
                        <el-option label="05" value="05"></el-option>
                        <el-option label="10" value="10"></el-option>
                    </el-select>
                    <div style="display: inline;padding: 5px;margin-left: -4px;border: 1px solid #e9e9e9;background: #F5F7FA;"><%=rb.getString("ANRFenZhong")%></div>
                </el-form-item>
                <span v-if="cllectExisted">
                    <span style="color: #B3B3B3;"><%=rb.getString("ShouJiBaoWenFuGaiTiShi")%> SN={{existedMsgSN}}. </span>
                </span>
            </el-form>

            <div slot="footer" style="text-align: right;">
                <el-button type="primary" @click="sendCollect"><%=rb.getString("QueDing")%></el-button>
                <el-button @click="collectMessageShow = false"><%=rb.getString("QuXiao")%></el-button>
            </div>
        </el-dialog>
        
        <el-dialog title="<%=rb.getString("YiDongDaoSheBeiZu")%>" :visible.sync="deviceGroupMoveShow" width="620"
			:close-on-click-modal="false" top="30vh" :append-to-body="true">
            <el-ctable ref="ctableGroup" :url="deviceGroupUrl" id='ctableGroup' @row-click="groupIdChange" :height="groupHeight" :row-key="'id'" pagination="true" :rownumber=true style='border: 1px solid #E9E9E9;'>
				<el-table-column label='' width="40">
					<template slot-scope="scope">
		           		<el-radio v-model="groupId" :label="scope.row.id"><span></span></el-radio>
		         	</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("SheBeiZuMingCheng")%>' min-width="150" prop="group_name"></el-table-column>
			</el-ctable>
			<div v-show="deviceGroupTip"><div slot="tip" class="el-upload__tip" ><%=rb.getString("QingXuanZeSheBeiZu")%> </div></div>
            <div slot="footer" style='text-align: right;'>
                <el-button type="primary" @click="deviceGroupMoveSubmit"><%=rb.getString("QueDing")%></el-button>
                <el-button @click="deviceGroupMoveShow = false"><%=rb.getString("QuXiao")%></el-button>
            </div>
        </el-dialog>
		<!--  锁定图标点击-->
        <el-dialog title="<%=rb.getString("QueRenNew")%>" :visible.sync="lockMessageShow" width="620" class="lockReasonDialog"
			:close-on-click-modal="false" top="30vh" :append-to-body="true">
            <el-form ref="lockForm" :model="lockForm">
                <div>
					<span v-show='lockReasonShow' style='display: block; color: #3D3D3D; font-size: 14px; word-wrap: break-word;'><%=rb.getString("SuoDingYuanYin")%>： {{lockReason}}</span>
					<span style='display: block; color: #3D3D3D; font-size: 14px; margin-top: 10px;'><%=rb.getString("SuoDingMac")%>： {{lockMac}}</span>
				</div>
                <el-form-item label=' ' prop='lockStatus' style='margin-top: 25px;'>
					<el-radio-group v-model='lockForm.lockStatus'>
						<el-radio label="0"><%=rb.getString("GuanBiSuoDingMAC")%></el-radio>
                        <el-radio label="1" style='margin-left: 60px;'><%=rb.getString("ZiDongTanCeJiSuoDing")%></el-radio>
					</el-radio-group>
				</el-form-item>
            </el-form>
            <div slot="footer" style='text-align: right;'>
                <el-button type="primary" @click="lockSubmit" v-loading="addLoading"><%=rb.getString("QueDing")%></el-button>
                <el-button @click="lockMessageShow = false"><%=rb.getString("QuXiao")%></el-button>
            </div>
        </el-dialog>

        <!-- 参数同步弹窗 -->
        <el-dialog title="<%=rb.getString("TongBu")%>" :visible.sync="openSyncDialog" :close-on-click-modal="false" @close="closeSyncDialog">
            <div id="sync-content" :class="{'loading': isSyncloading}" style="min-height: 200px;">
            </div>
            
        </el-dialog>
    </div>

    <!-- Limitation 功能 -->
    <%@ include file="limitation.jsp" %>

    <script>
        // vxe-table 注册到 Vue
        var VxeTable = window.VXETable || window.VxeTable || window.VxeUITable;
        Vue.use(VxeTable);
        
        // vxe-table 全局配置
        if (VxeTable && VxeTable.setup) {
            VxeTable.setup({
                table: {
                    showOverflow: true,
                    showHeaderOverflow: true,
                    border: true,
                    size: 'mini'
                }
            });
        }
        
        var enbShowCols = '${showCol}',
            columncell = "${cellColumn}",//未选中的列表标识
            allColumn = "${allColumn}",
            eNodeB_column =[],
            enableCheckbox =  (writableMap['CODE_ENB_REBOOT'] == true || writableMap['CODE_ENB_SYNCHRONIZE'] == true || writableMap['CODE_ENB_MONITOR'] == true),
            is_reboot_batch = false,
            halobSwitchFlag = '';

        $('.el-popover:has(#enb_export_content)').remove();

        var enbvm = new Vue({
            el: '#cellInfo',
            components: {
                draggable: window.vuedraggable || window.Vuedraggable
            },
            data() {
                var vm = this;

                return {
                    // vxe-table数据和状态
                    tableData: [],
                    tableHeight: '100%',
                    tableLoading: false,
                    tableKey: 0, // 用于强制重新渲染表格
                    
                    // 分页配置
                    pager: {
                        currentPage: 1,
                        pageSize: 50,
                        total: 0
                    },
                    
                    // Remark label 编辑状态
                    editingRemarkLabel: false,
                    remarkLabelInput: 'Remark',
                    currentRemarkLabel: 'Remark',

                    isSyncloading:false,

                    cllectExisted: false,
                    existedMsgSN: '',
                    collectMessageShow: false,
                    collectForm: {
                        collectInterval: ''
                    },

                    // 列配置相关数据
                    columnConfigVisible: false,
                    columnConfigDrag: false,
                    // 列分类数据（与导出页面export_config.jsp保持一致）
                    deviceCol: [
                        {code: 'serial_number', label: '<%=rb.getString("XiaoZhanBianMa")%>', disabled: true},
                        {code: 'product', label: '<%=rb.getString("ChanPinLeiXingBiaoZhi")%>', disabled: true},
                        {code: 'product_name', label: '<%=rb.getString("ChanPinMingCheng")%>'},
                        {code: 'module_type', label: '<%=rb.getString("SheBeiXingHaoMing")%>', disabled: true},
                        {code: 'software_version', label: '<%=rb.getString("SoftwareVersion")%>', disabled: true},
                        {code: 'firmware_version', label: '<%=rb.getString("FirmwareVersion")%>'},
                        {code: 'online_duration', label: '<%=rb.getString("LeiJiShiChang")%>'},
                        {code: 'up_time', label: '<%=rb.getString("YunXingShiJian")%>'},
                        {code: 'first_online_time', label: '<%=rb.getString("DiYiCiLianJieShiJian")%>'},
                        {code: 'LASTINFORMTIME', label: '<%=rb.getString("ShangCiLianJieShiJian")%>'},
                        {code: 'online_time', label: '<%=rb.getString("JieRuShiJian")%>'},
                        {code: 'offline_time', label: '<%=rb.getString("DuanKaiShiJian")%>'},
                        {code: 'mac_address', label: '<%=rb.getString("MACDiZhi")%>', disabled: true},
                        {code: 'gps_version', label: '<%=rb.getString("GPSBanBen")%>'},
                        {code: 'group_name', label: '<%=rb.getString("SheBeiZu")%>', disabled: true},
                        {code: 'sub_station_name', label: siteNameLabel},
                        {code: 'rom', label: 'Rom'},
                        {code: 'remark', label: 'Remark'}
                    ],
                    cellCol: [
                        {code: 'enbId', label: '<%=rb.getString("EnodebId")%>'},
                        {code: 'host_name', label: '<%=rb.getString("HostName")%>', disabled: true},
                        {code: 'cellId', label: '<%=rb.getString("XIAOQUID")%>'},
                        {code: 'CELL_IDENTITY', label: 'ECI', disabled: true},
                        {code: 'PHYCELLID', label: '<%=rb.getString("PCI2")%>', disabled: true},
                        {code: 'plmnid', label: '<%=rb.getString("PLMN")%>'},
                        {code: 'tac', label: '<%=rb.getString("TAC")%>'},
                        {code: 'signment', label: '<%=rb.getString("ZiZhenPeiBi")%>'},
                        {code: 'specialSubframe', label: '<%=rb.getString("TeShuZiZhenPeiBi")%>'},
                        {code: 'rootIndex', label: '<%=rb.getString("GenXuLieSuoYin")%>'},
                        {code: 'site_id', label: siteIdLabel},
                        {code: 'bandwidth', label: '<%=rb.getString("DaiKuan")%>'},
                        {code: 'EARFCNDLINUSE', label: '<%=rb.getString("PinDian")%>'},
                        {code: 'network_model', label: '<%=rb.getString("JiZhanZhiShi")%>'},
                        {code: 'tx_power', label: '<%=rb.getString("CPETxPower")%>'}
                    ],
                    statusCol: [
                        {code: 'op_state', label: '<%=rb.getString("ShiFouJiHuo")%>', disabled: true},
                        {code: 'mme_status', label: '<%=rb.getString("MMEZhuangTai")%>', disabled: true},
                        {code: 'rf_status', label: '<%=rb.getString("ShePinKaiGuanZhuangTai")%>', disabled: true},
                        {code: 'pm_report_status', label: '<%=rb.getString("KPIShangBaoZhuangTai")%>'},
                        {code: 'halob_flag', label: 'HaloX'},
                        {code: 'synStatus', label: '<%=rb.getString("TongBuZhuangTai")%>'},
                        {code: 'validity', label: '<%=rb.getString("YouXiaoQi")%>'},
                        {code: 'lock_status', label: '<%=rb.getString("SuoDingZhuangTai")%>'},
                        {code: 'ue_count', label: '<%=rb.getString("UEShu")%>', disabled: true},
                        {code: 'euCountStr', label: '<%=rb.getString("EUShu")%>'},
                        {code: 'ruCountStr', label: '<%=rb.getString("RUShu")%>'},
                        {code: 'cpe_connect', label: '<%=rb.getString("CPELianJieShu")%>', disabled: true},
                        {code: 'wanSpeed', label: '<%=rb.getString("WanZhuangTai")%>'},
                        {code: 'service_status', label: '<%=rb.getString("ZhuangTai")%>'}
                    ],
                    networkCol: [
                        {code: 'mmepool_ipsec_addr', label: '<%=rb.getString("MMEPoolIPSECDiZhi")%>'},
                        {code: 'cell_ip', label: '<%=rb.getString("IPDiZhi")%>', disabled: true},
                        {code: 'IPSEC_ADDR', label: '<%=rb.getString("IPSECDiZhi")%>'}
                    ],
                    locationCol: [
                        {code: 'gps_longitude', label: '<%=rb.getString("GPSJingDu")%>'},
                        {code: 'gps_latitude', label: '<%=rb.getString("GPSWeiDu")%>'},
                        {code: 'gps_height', label: '<%=rb.getString("GPSGaoDu")%>'},
                        {code: 'mechanical_downtilt', label: '<%=rb.getString("JiXieXiaQingJiao")%>'},
                        {code: 'electronic_downtilt', label: '<%=rb.getString("DianZiXiaQingJiao")%>'},
                        {code: 'vertical_3dB_beam_width', label: '<%=rb.getString("ChuiZhiBoSuKuanDu")%>'},
                        {code: 'horizontal_azimuth', label: '<%=rb.getString("ShuiPinFangWeiJiao")%>'},
                        {code: 'install_address', label: '<%=rb.getString("AnZhuangXiangXiDiZhi")%>'}
                    ],
                    satelliteCol: [
                        {code: 'gps_satellite_count', label: '<%=rb.getString("GPSWeiXingShu")%>'}
                    ],
                    // 列配置表单
                    columnForm: {
                        device: ['serial_number', 'product', 'module_type', 'software_version', 'mac_address', 'group_name', 'online_time', 'offline_time'],
                        cell: ['host_name', 'CELL_IDENTITY', 'PHYCELLID'],
                        status: ['op_state', 'mme_status', 'rf_status', 'ue_count', 'cpe_connect'],
                        network: ['cell_ip'],
                        location: [],
                        satellite: []
                    },
                    // 列配置展开状态
                    columnExpanded: {
                        device: true,
                        cell: true,
                        status: true,
                        network: true,
                        location: true,
                        satellite: true
                    },
                    // 拖拽列排序
                    dragColumns: [],

                    sortColumns: [],
                    columns: [
                        {field: 'serial_number', label: '<%=rb.getString("XiaoZhanBianMa")%>',sortable: true, width: 180, disabled: true},
                        {field: 'enbId', label: '<%=rb.getString("EnodebId")%>', width: 100},
                        {field: 'host_name', label: '<%=rb.getString("HostName")%>',sortable: true, width: 150, disabled: true},
                        {field: 'cellId', label: '<%=rb.getString("XIAOQUID")%>', width: 100},
                        {field: 'rf_status', label: '<%=rb.getString("ShePinKaiGuanZhuangTai")%>',sortable: true, width: 150, disabled: true},
                        {field: 'op_state', label: '<%=rb.getString("ShiFouJiHuo")%>',sortable: true, width: 120, disabled: true},
                        {field: 'CELL_IDENTITY', label: 'ECI',sortable: true, width: 100, disabled: true},
                        {field: 'PHYCELLID', label: '<%=rb.getString("PCI2")%>',sortable: true, width: 70, disabled: true},
                        {field: 'mme_status', label: '<%=rb.getString("MMEZhuangTai")%>', width: 140, disabled: true},
                        {field: 'plmnid', label: '<%=rb.getString("PLMN")%>', width: 70},
                        {field: 'bandwidth', label: '<%=rb.getString("DaiKuan")%>', width: 80},
                        {field: 'ue_count', label: '<%=rb.getString("UEShu")%>',sortable: true, width: 80},
                        {field: 'euCountStr', label: '<%=rb.getString("EUShu")%>',sortable: true, width: 80},
                        {field: 'ruCountStr', label: '<%=rb.getString("RUShu")%>',sortable: true, width: 80},
                        {field: 'cpe_connect', label: '<%=rb.getString("CPELianJieShu")%>', width: 100},
                        {field: 'wanSpeed', label: '<%=rb.getString("WanZhuangTai")%>', width: 180},
                        {field: 'cell_ip', label: '<%=rb.getString("IPDiZhi")%>',sortable: true, width: 120},
                        {field: 'mac_address', label: '<%=rb.getString("MACDiZhi")%>',sortable: true, width: 130},
                        {field: 'product', label: '<%=rb.getString("ChanPinLeiXingBiaoZhi")%>',sortable: true, width: 110},
                        {field: 'product_name', label: '<%=rb.getString("ChanPinMingCheng")%>',sortable: true, width: 120},
                        {field: 'module_type', label: '<%=rb.getString("SheBeiXingHaoMing")%>',sortable: true, width: 120},
                        {field: 'software_version', label: '<%=rb.getString("SoftwareVersion")%>',sortable: true, width: 140},
                        {field: 'group_name', label: '<%=rb.getString("SheBeiZu")%>',sortable: true, width: 130},
                        {field: 'IPSEC_ADDR', label: '<%=rb.getString("IPSECDiZhi")%>', width: 170},
                        {field: 'mmepool_ipsec_addr', label: '<%=rb.getString("MMEPoolIPSECDiZhi")%>', width: 120},
                        {field: 'site_id', label: siteIdLabel,sortable: true, width: 80},

                        {field: 'sub_station_name', label: siteNameLabel,width: 120},
                        {field: 'install_address', label: '<%=rb.getString("AnZhuangXiangXiDiZhi")%>', width: 200},
                        {field: 'service_status', label: '<%=rb.getString("ZhuangTai")%>',width: 120},
                        {field: 'rom', label: 'Rom', width: 120},
                        
                        {field: 'EARFCNDLINUSE', label: '<%=rb.getString("PinDian")%>', width: 130},
                        {field: 'synStatus', label: '<%=rb.getString("TongBuZhuangTai")%>',sortable: true, width: 160},
                        {field: 'pm_report_status', label: '<%=rb.getString("KPIShangBaoZhuangTai")%>',sortable: true, width: 135},
                        {field: 'gps_satellite_count', label: '<%=rb.getString("GPSWeiXingShu")%>',sortable: true, width: 85},
                        {field: 'online_duration', label: '<%=rb.getString("LeiJiShiChang")%>',sortable: true, width: 120},
                        {field: 'up_time', label: '<%=rb.getString("YunXingShiJian")%>',sortable: true, width: 120},
                        {field: 'first_online_time', label: '<%=rb.getString("DiYiCiLianJieShiJian")%>',sortable: true, width: 140},
                        {field: 'LASTINFORMTIME', label: '<%=rb.getString("ShangCiLianJieShiJian")%>',sortable: true, width: 140},
                        {field: 'online_time', label: '<%=rb.getString("JieRuShiJian")%>',sortable: true, width: 140},
                        {field: 'offline_time', label: '<%=rb.getString("DuanKaiShiJian")%>',sortable: true, width: 140},
                        {field: 'network_model', label: '<%=rb.getString("JiZhanZhiShi")%>',sortable: true, width: 120},
                        {field: 'firmware_version', label: '<%=rb.getString("FirmwareVersion")%>',sortable: true, width: 135},
                        {field: 'gps_version', label: '<%=rb.getString("GPSBanBen")%>', width: 120},
                        {field: 'available_rate', label: '<%=rb.getString("XiaoQuKeYongZhanBi")%>', width: 70},
                        {field: 'halob_flag', label: 'HaloX',sortable: true, width: 100},
                        {field: 'tac', label: '<%=rb.getString("TAC")%>', width: 80},
                        {field: 'signment', label: '<%=rb.getString("ZiZhenPeiBi")%>', width: 80},
                        {field: 'specialSubframe', label: '<%=rb.getString("TeShuZiZhenPeiBi")%>', width: 80},
                        {field: 'rootIndex', label: '<%=rb.getString("GenXuLieSuoYin")%>', width: 80},
                        {field: 'gps_longitude', label: '<%=rb.getString("GPSJingDu")%>', width: 100},
                        {field: 'gps_latitude', label: '<%=rb.getString("GPSWeiDu")%>', width: 90},
                        {field: 'gps_height', label: '<%=rb.getString("GPSGaoDu")%>', width: 70},
                        
                        {field: 'mechanical_downtilt', label: '<%=rb.getString("JiXieXiaQingJiao")%>', width: 135},
                        {field: 'electronic_downtilt', label: '<%=rb.getString("DianZiXiaQingJiao")%>', width: 135},
                        {field: 'vertical_3dB_beam_width', label: '<%=rb.getString("ChuiZhiBoSuKuanDu")%>', width: 160},
                        {field: 'horizontal_azimuth', label: '<%=rb.getString("ShuiPinFangWeiJiao")%>', width: 135},
                        
                        {field: 'validity', label: '<%=rb.getString("YouXiaoQi")%>', width: 125},
                        {field: 'lock_status', label: '<%=rb.getString("SuoDingZhuangTai")%>', width: 90},
                        {field: 'tx_power', label: '<%=rb.getString("CPETxPower")%>', width: 80},
                        {field: 'remark', label: vm.currentRemarkLabel, width: 140},
                        // {field: 'authCode', label: '<%=rb.getString("JianQuanMa")%>', width: 120}
                    ],
                    activeName: 'eNB',
                    selectedRow: '',
                    menus: [],
                    settingUrl:'',
                    tbURL: '${ctx}/cell/cpeinfos/queryCpeInfosList.action?monitor=1',
                    queryParams: {
                        sort: '',
                        order: '',
                        rows: 50,
                        page: 1,
                        TimeZone : timeZone,
                        isDual: false,
                        isMonitor: true,
                        search_text: '',
                        like_fields: 'serial_number,host_name,cell_ip',
                        connection_status: [],
						op_state: '',
						product_model: [],
						model_name: [],
						software_version: [],
						firmware_version: [],
						halob_flag: '',
						group_id: [],
                        halodSerialNumbers:''
                    },

                    showProps: enbShowCols.split(','),
                    enableCheckbox: (writableMap['CODE_ENB_REBOOT'] == true || writableMap['CODE_ENB_SYNCHRONIZE'] == true ||  writableMap['CODE_ENB_MONITOR'] == true),
                    selectedRows: [],

                    infoslide: {
                        title: '<%=rb.getString("XinXi")%>',
                        url: ''
                    },
                    ueslide: {
                        title: '',
                        height: '',
                        data: '',
                        is436q: false
                    },
                    cpeslide: {
                        title: '',
                        url: '',
                    },
                    activeslide: {
                        title: '<%=rb.getString("XiQuKeYongXiangQing")%>',
                        data: ''
                    },
                    periodslide: {
                        title: '',
                        url: ''
                    },
                    distributeslide: {
                        title: '',
                        url: ''
                    },
                    enbSettingSlide: {
                        title: '',
                        url: ''
                    },
                    satelliteURL: '',
                    sateloading: true,
                    activeCells:[],
                    cellInfosList:[],
                    showActiveCellDialog:false,
                    activeCellLialogLoading:false,
                    activeCellsDataList:[],
                    cellActivePopoverLoading:false,
                    deviceGroupMoveShow: false,
                    groupId: '',
                    deviceGroupUrl:'${ctx}/pm/template/getTempDeviceGroupList.action',
            		groupHeight:'320px',
            		deviceGroupTip:false,
                    halodCellPopoverLoading:false,
                    halodCellDataList:[],
		            lockMessageShow: false,
                    lockForm: {
                    	lockStatus: '0',
                    },
                    lockReason: '',
                    lockMac: '',
                    curRowSn: '',
                    addLoading: false,
                    lockReasonShow: false,
                    bulkSelectShow:false,
                    monitorHeadType:'',
                    monitorHeadBtnData:[],
                    
                    ueS1apId: false,
                    mmeS1apId: false,

                    openSyncDialog:false,
                    enbAdditionalColShow:'${enbAdditionalColShow}',
                    batchSync:false,
                    batchCode:'',
                    gsmEnable: '${gsmEnable}',
                    platformType: '',
                };
            },
            computed: {
                limitBatch(){
                    return batchOperation ? '' : 1;
                },
                confirmTips() {
                    var vm = this,
                        sn = vm.selectedRow.serial_number,
                        msg = '<%=rb.getString("QueRenShouJiPre")%>';

                    return msg.replace('placeholder', sn);
                },
                showColums() {
                    var vm = this,
                        columns = vm.getAllDefaultCols(),
                        props = columns.map(function(col){
                            return col.prop;
                        });

                    props = props.filter(function(code){
                        return vm.showProps.includes(code);
                    });

                    return props;
                },
                isMonitorWritable() {
                    return writableMap.CODE_ENB_MONITOR == true;
                },
                // 排序后的可见列配置
                sortedVisibleColumns() {
                    var vm = this,
                        fixedCols = [
                            {type: 'seq', key: 'col_seq'},
                            {type: 'checkbox', key: 'col_checkbox', visible: vm.enableCheckbox},
                            {field: 'enb_operation', key: 'col_enb_operation'},
                            {field: 'connection_status', key: 'col_connection_status'},
                            {field: 'alarm', key: 'col_alarm'},
                            {field: 'serial_number', key: 'col_serial_number'},
                            {field: 'host_name', key: 'col_host_name'}
                        ],
                        sortableCols = vm.columns.filter(function(col) {
                            // 排除固定列
                            var fixedFields = ['serial_number', 'host_name'];
                            return !fixedFields.includes(col.field);
                        }),
                        showCols = vm.showProps || [],
                        sortOrder = vm.sortColumns || [];
                    
                    // 过滤出需要显示的列
                    sortableCols = sortableCols.filter(function(col) {
                        return showCols.includes(col.field);
                    });
                    
                    // 按 sortColumns 排序
                    sortableCols.sort(function(a, b) {
                        var idxA = sortOrder.indexOf(a.field),
                            idxB = sortOrder.indexOf(b.field);
                        idxA = idxA === -1 ? 999 : idxA;
                        idxB = idxB === -1 ? 999 : idxB;
                        return idxA - idxB;
                    });
                    
                    // 为每个列添加 key
                    sortableCols = sortableCols.map(function(col, index) {
                        return Object.assign({}, col, {key: 'col_' + col.field + '_' + index});
                    });
                    
                    // 过滤固定列（复选框列需要根据 enableCheckbox 判断）
                    fixedCols = fixedCols.filter(function(col) {
                        if (col.type === 'checkbox') {
                            return col.visible;
                        }
                        return true;
                    });
                    
                    return fixedCols.concat(sortableCols);
                },
                // 列配置相关计算属性
                selectedColumnCodes() {
                    var vm = this;
                    return vm.columnForm.device
                        .concat(vm.columnForm.cell)
                        .concat(vm.columnForm.status)
                        .concat(vm.columnForm.network)
                        .concat(vm.columnForm.location)
                        .concat(vm.columnForm.satellite);
                },
                isColumnIndeterminate() {
                    var vm = this,
                        total = vm.filteredDeviceCol.length + vm.filteredCellCol.length + vm.filteredStatusCol.length + 
                                vm.filteredNetworkCol.length + vm.filteredLocationCol.length + vm.satelliteCol.length,
                        selected = vm.selectedColumnCodes.length;
                    return selected > 0 && selected < total;
                },
                isColumnAllSelected: {
                    get: function() {
                        var vm = this,
                            total = vm.filteredDeviceCol.length + vm.filteredCellCol.length + vm.filteredStatusCol.length + 
                                    vm.filteredNetworkCol.length + vm.filteredLocationCol.length + vm.satelliteCol.length;
                        return vm.selectedColumnCodes.length >= total;
                    },
                    set: function(val) {}
                },
                isDeviceAllSelected: {
                    get: function() { return this.columnForm.device.length >= this.filteredDeviceCol.length; },
                    set: function(val) {}
                },
                isCellAllSelected: {
                    get: function() { return this.columnForm.cell.length >= this.filteredCellCol.length; },
                    set: function(val) {}
                },
                isStatusAllSelected: {
                    get: function() { return this.columnForm.status.length >= this.filteredStatusCol.length; },
                    set: function(val) {}
                },
                isNetworkAllSelected: {
                    get: function() { return this.columnForm.network.length >= this.filteredNetworkCol.length; },
                    set: function(val) {}
                },
                isLocationAllSelected: {
                    get: function() { return this.columnForm.location.length >= this.filteredLocationCol.length; },
                    set: function(val) {}
                },
                isSatelliteAllSelected: {
                    get: function() { return this.columnForm.satellite.length >= this.satelliteCol.length; },
                    set: function(val) {}
                },
                // 根据 getAllDefaultCols 条件过滤后的列配置（用于列设置浮层）
                filteredLocationCol() {
                    var vm = this,
                        allCols = vm.getAllDefaultCols(),
                        allColProps = allCols.map(function(c) { return c.prop; });
                    return vm.locationCol.filter(function(item) {
                        return allColProps.includes(item.code);
                    });
                },
                filteredNetworkCol() {
                    var vm = this,
                        allCols = vm.getAllDefaultCols(),
                        allColProps = allCols.map(function(c) { return c.prop; });
                    return vm.networkCol.filter(function(item) {
                        return allColProps.includes(item.code);
                    });
                },
                filteredDeviceCol() {
                    var vm = this,
                        allCols = vm.getAllDefaultCols(),
                        allColProps = allCols.map(function(c) { return c.prop; });
                    return vm.deviceCol.filter(function(item) {
                        return allColProps.includes(item.code);
                    });
                },
                filteredStatusCol() {
                    var vm = this,
                        allCols = vm.getAllDefaultCols(),
                        allColProps = allCols.map(function(c) { return c.prop; });
                    return vm.statusCol.filter(function(item) {
                        return allColProps.includes(item.code);
                    });
                },
                filteredCellCol() {
                    var vm = this,
                        allCols = vm.getAllDefaultCols(),
                        allColProps = allCols.map(function(c) { return c.prop; });
                    return vm.cellCol.filter(function(item) {
                        return allColProps.includes(item.code);
                    });
                }
            },
            watch:{
                selectedRows(newVal){
                    if(newVal.length == 0){
                        this.bulkSelectShow = false;
                    }
                },
                queryParams: {
                    handler: function(newVal, oldVal) {
                        var vm = this;
                        // 查询参数变化时，重置到第一页
                        vm.pager.currentPage = 1;
                        vm.loadTableData();
                    },
                    deep: true
                }
            },
            methods: {
                // ==================== Tooltip 相关方法 ====================
                // 自定义单元格 tooltip 内容，解决刚好溢出时不显示的问题
                cellTooltipMethod({ type, column, row, $event }) {
                    // 表头使用默认行为
                    if (type === 'header') {
                        return null;
                    }
                    // 跳过有自定义模板的特殊列（这些列有自己的 popover）
                    var skipFields = ['plmnid', 'enb_operation', 'connection_status', 'alarm', 'host_name', 'mme_info', 'halod_cell', 'run_status', 'sync_status'];
                    if (!column || !column.field || skipFields.includes(column.field)) {
                        return null;
                    }
                    // 获取单元格值
                    if (!row) return null;
                    var cellValue = row[column.field];
                    if (cellValue === null || cellValue === undefined || cellValue === '') {
                        return null;
                    }
                    var text = String(cellValue);
                    // 获取触发事件的单元格元素
                    var cell = $event ? $event.currentTarget : null;
                    if (cell) {
                        var cellEl = cell.querySelector('.vxe-cell');
                        if (cellEl) {
                            // 使用更宽松的判断：如果文本宽度接近容器宽度就显示 tooltip
                            if (cellEl.scrollWidth >= cellEl.clientWidth - 2) {
                                return text;
                            }
                        }
                    }
                    // 备用判断：基于列宽和文本长度估算
                    var colWidth = column.renderWidth || 100;
                    var avgCharWidth = 8;
                    var textWidth = text.length * avgCharWidth;
                    if (textWidth >= colWidth - 20) {
                        return text;
                    }
                    return null;
                },
                // ==================== 列配置相关方法 ====================
                // 打开列配置对话框
                openColumnConfig() {
                    var vm = this;
                    // 初始化 columnForm 从 showProps
                    vm.initColumnFormFromShowProps();
                    // 初始化拖拽列
                    vm.initDragColumns();
                    //vm.columnConfigVisible = true;
                },
                // 从 showProps 初始化 columnForm
                initColumnFormFromShowProps() {
                    var vm = this,
                        showProps = vm.showProps;
                    
                    // 筛选显示的列，确保disabled的列始终被选中，并只包含filteredXxxCol中的列
                    vm.columnForm.device = vm.filteredDeviceCol.filter(function(item) {
                        return item.disabled || showProps.includes(item.code);
                    }).map(function(item) { return item.code; });
                    
                    vm.columnForm.cell = vm.filteredCellCol.filter(function(item) {
                        return item.disabled || showProps.includes(item.code);
                    }).map(function(item) { return item.code; });
                    
                    vm.columnForm.status = vm.filteredStatusCol.filter(function(item) {
                        return item.disabled || showProps.includes(item.code);
                    }).map(function(item) { return item.code; });
                    
                    vm.columnForm.network = vm.filteredNetworkCol.filter(function(item) {
                        return item.disabled || showProps.includes(item.code);
                    }).map(function(item) { return item.code; });
                    
                    vm.columnForm.location = vm.filteredLocationCol.filter(function(item) {
                        return item.disabled || showProps.includes(item.code);
                    }).map(function(item) { return item.code; });
                    
                    vm.columnForm.satellite = vm.satelliteCol.filter(function(item) {
                        return item.disabled || showProps.includes(item.code);
                    }).map(function(item) { return item.code; });
                },
                // 初始化拖拽列
                initDragColumns() {
                    var vm = this,
                        sortCols = vm.sortColumns.length > 0 ? vm.sortColumns : vm.columns.map(function(c) { return c.field; });
                    
                    // 根据 sortColumns 排序 columns
                    vm.dragColumns = vm.columns.slice().sort(function(a, b) {
                        var idxA = sortCols.indexOf(a.field);
                        var idxB = sortCols.indexOf(b.field);
                        idxA = idxA === -1 ? 100 : idxA;
                        idxB = idxB === -1 ? 100 : idxB;
                        return idxA - idxB;
                    });
                },
                // 应用列配置
                applyColumnConfig() {
                    var vm = this,
                        url = '${ctx}/system/column/setting/insert.action',
                        showCols = vm.selectedColumnCodes,
                        sortCol = vm.dragColumns.map(function(item) { return item.field; });
                    
                    // 更新 showProps
                    vm.showProps = showCols;
                    // 更新 sortColumns
                    vm.sortColumns = sortCol;
                    
                    // 保存到后端
                    var params = {
                        pageName: '1',
                        showColumn: showCols.join(','),
                        sortColumn: sortCol.join(',')
                    };
                    
                    axios.post(url, params).then(function(res) {
                        vm.$message.success('<%=rb.getString("ChengGong")%>');
                        vm.columnConfigVisible = false;
                        // 使用 vxe-table 的列排序功能
                        vm.$nextTick(function() {
                            vm.reorderTableColumns();
                        });
                    }).catch(function(err) {
                        vm.$message.error('<%=rb.getString("ShiBai")%>');
                    });
                },
                // 重新排序表格列 - 通过更新 tableKey 强制重新渲染
                reorderTableColumns() {
                    var vm = this;
                    // 更新 tableKey 强制 vxe-table 重新渲染
                    vm.tableKey++;
                    console.log('reorderTableColumns - tableKey updated to:', vm.tableKey);
                    console.log('reorderTableColumns - sortColumns:', vm.sortColumns);
                    console.log('reorderTableColumns - showProps:', vm.showProps);
                },
                // 全选/取消全选
                handleColumnAllChange(val) {
                    var vm = this;
                    vm.handleDeviceAllChange(val);
                    vm.handleCellAllChange(val);
                    vm.handleStatusAllChange(val);
                    vm.handleNetworkAllChange(val);
                    vm.handleLocationAllChange(val);
                    vm.handleSatelliteAllChange(val);
                },
                handleDeviceAllChange(val) {
                    var vm = this;
                    if (val) {
                        vm.columnForm.device = vm.filteredDeviceCol.map(function(item) { return item.code; });
                    } else {
                        vm.columnForm.device = vm.filteredDeviceCol.filter(function(item) { return item.disabled; }).map(function(item) { return item.code; });
                    }
                },
                handleCellAllChange(val) {
                    var vm = this;
                    if (val) {
                        vm.columnForm.cell = vm.filteredCellCol.map(function(item) { return item.code; });
                    } else {
                        vm.columnForm.cell = vm.filteredCellCol.filter(function(item) { return item.disabled; }).map(function(item) { return item.code; });
                    }
                },
                handleStatusAllChange(val) {
                    var vm = this;
                    if (val) {
                        vm.columnForm.status = vm.filteredStatusCol.map(function(item) { return item.code; });
                    } else {
                        vm.columnForm.status = vm.filteredStatusCol.filter(function(item) { return item.disabled; }).map(function(item) { return item.code; });
                    }
                },
                handleNetworkAllChange(val) {
                    var vm = this;
                    if (val) {
                        vm.columnForm.network = vm.filteredNetworkCol.map(function(item) { return item.code; });
                    } else {
                        vm.columnForm.network = vm.filteredNetworkCol.filter(function(item) { return item.disabled; }).map(function(item) { return item.code; });
                    }
                },
                handleLocationAllChange(val) {
                    var vm = this;
                    if (val) {
                        vm.columnForm.location = vm.filteredLocationCol.map(function(item) { return item.code; });
                    } else {
                        vm.columnForm.location = vm.filteredLocationCol.filter(function(item) { return item.disabled; }).map(function(item) { return item.code; });
                    }
                },
                handleSatelliteAllChange(val) {
                    var vm = this;
                    if (val) {
                        vm.columnForm.satellite = vm.satelliteCol.map(function(item) { return item.code; });
                    } else {
                        vm.columnForm.satellite = vm.satelliteCol.filter(function(item) { return item.disabled; }).map(function(item) { return item.code; });
                    }
                },
                // 从表单中移除列
                removeColumnFromForm(code) {
                    var vm = this;
                    ['device', 'cell', 'status', 'network', 'location', 'satellite'].forEach(function(group) {
                        var idx = vm.columnForm[group].indexOf(code);
                        if (idx > -1) {
                            vm.columnForm[group].splice(idx, 1);
                        }
                    });
                },
                // ==================== 列配置方法结束 ====================

                // 获取自定义label信息
                getCustomLabelData() {
                    var vm = this;

                    axios.post('${ctx}/cell/columnAlias/queryColumnAliasConfigs.action').then(function(response){
                        var data = response.data || [];

                        data.forEach(function(item){
                            if(item.columnName == 'remark'){
                                vm.currentRemarkLabel = item.columnAlias || 'Remark';
                                vm.remarkLabelInput = vm.currentRemarkLabel;
                            }
                        });
                    }).catch(function(error){});
                },
                // Remark label 编辑方法
                startEditRemarkLabel() {
                    this.editingRemarkLabel = true;
                    this.remarkLabelInput = this.currentRemarkLabel;
                },
                saveRemarkLabel() {
                    var vm = this,
                        params = {
                            columnName: 'remark',
                            columnAlias: this.remarkLabelInput.trim()
                        };
                    if(this.remarkLabelInput.trim() === ''){
                        vm.$message.warning('<%=rb.getString("QingShuRuBiTianXiang")%>');
                        return;
                    }
                    let paramsData = JSON.stringify(params);
                    axios.post('${ctx}/cell/columnAlias/setColumnAlias.action',paramsData,{headers:{'Content-Type':'application/json;charset=utf-8'}}).then(function(response){
                        var data = response.data;

                        if(data.success){
                            vm.currentRemarkLabel = vm.remarkLabelInput.trim();
                            vm.editingRemarkLabel = false;
                            vm.syncRemarkLabel();
                            vm.$message.success('<%=rb.getString("ChengGong")%>');
                        }else{
                            vm.$message.error(data.message);
                        }
                    }).catch(function(error){});
                },
                // 取消编辑 Remark label
                cancelEditRemarkLabel() {
                    this.editingRemarkLabel = false;
                    this.remarkLabelInput = this.currentRemarkLabel;
                },
                // 同步其他页面的Remark label
                syncRemarkLabel() { 
                    var vm = this,
                        newLabel = vm.currentRemarkLabel,
                        vueInstances = [
                            {name: 'gsmvm', instance: typeof gsmvm !== 'undefined' ? gsmvm : null},
                            {name: 'gnbMonitor', instance: typeof gnbMonitor !== 'undefined' ? gnbMonitor : null},
                            {name: 'egwRegisterVue', instance: typeof egwRegisterVue !== 'undefined' ? egwRegisterVue : null},
                            {name: 'gnbOverviewVue', instance: typeof gnbOverviewVue !== 'undefined' ? gnbOverviewVue : null},
                            {name: 'enbDetailVue', instance: typeof enbDetailVue !== 'undefined' ? enbDetailVue : null},
                            {name: 'gsmSettingOverviewVue', instance: typeof gsmSettingOverviewVue !== 'undefined' ? gsmSettingOverviewVue : null},
                        ];

                    vueInstances.forEach(function(vue){
                        if(vue.instance && vue.instance.currentRemarkLabel !== undefined){
                            vue.instance.currentRemarkLabel = newLabel;
                        }
                        if(vue.instance && vue.instance.remarkLabelInput !== undefined){
                            vue.instance.remarkLabelInput = newLabel;
                        }
                    });
                },
                // 开关改变事件
                enabelChange(row) {
                    var vm = this,
                        confirmMsg='',
                        message='',
                        url='${ctx}/cell/cpeinfos/modifyIsCheckInActiveEnable.action',
                        params = {
                            isCheckInActive: row.isCheckInActive == '1' ? '0' : '1',
                            small_cell_code: row.small_cell_code
                        };

                    if(row.isCheckInActive === '0'){
                        confirmMsg = 'Are you sure you want to enable?' + 
                                     'eNb synchronize its latitude and longitude after reboot, if they do not match, eNB will be inactive.';
                        message = '<%=rb.getString("CaoZuoMingCheng")%>:<%=rb.getString("DaKaiSasKaiGuan")%>  <%=rb.getString("SheBeiMingCheng")%>'.replace('SAS','') + row.serial_number
                    }else{
                        confirmMsg = 'Are you sure you want to disable?';
                        message = '<%=rb.getString("CaoZuoMingCheng")%>:<%=rb.getString("GuanBiSasKaiGuan")%>  <%=rb.getString("SheBeiMingCheng")%>'.replace('SAS','') + row.serial_number
                    }

                    vm.$confirm(confirmMsg, '<%=rb.getString("QueRen")%>', {
                        customClass: 'warningConfirm',
                        confirmButtonText: '<%=rb.getString("QueDing")%>',
                        cancelButtonText: '<%=rb.getString("QuXiao")%>',
                        type: 'warning',
                        closeOnClickModal: false
                    }).then(() => {
                        vm.$refs[row.serial_number].activeIconClass= 'el-icon-loading';

                        axios.post(url,stringify(params)).then(function(response){
                            let data = response.data;

                            if(data.success){
                                vm.$notify({
                                    title: '<%=rb.getString("ChengGong")%>',
                                    message: message,
                                    type: 'success',
                                    duration: 10000,
                                })
                                vm.loadTableData();
                            }else{
                                vm.$notify({
                                    title: data.message,
                                    message: message,
                                    type: 'error',
                                    duration: 10000,
                                })
                            }

                            vm.$refs[row.serial_number].activeIconClass= '';
                        }).catch(function(error){ })
                    }).catch()

                    event.stopPropagation();
                },
                selectable(row, index) {
                    // 不是輔站的，才可被選中
                    if(row.dual_carrier_type == 2 && row.product != 'RTD') {
                        return false;
                    }else {
                        return true;
                    }
                },
                showCollectMessage(row) {
                    var vm = this,
                        paramsExist = {
                            type: 'enb',
                            operatorCode: operatorCodeGloab
                        };

                    axios.post('${ctx}/trace/isExistTracingDevice.action', stringify(paramsExist)).then(function(res){
                        var data = res.data;

                        if(data && data.isExist == 'true') {
                            vm.cllectExisted = true;
                            vm.existedMsgSN = data.serialNumber;
                        }else {
                            vm.cllectExisted = false;
                            vm.existedMsgSN = '';
                        }
                    });

                    vm.collectForm.collectInterval = '10';
                    vm.collectMessageShow = true;
                },
                sendCollect() {
                    var vm = this,
                        row = vm.selectedRow || {},
                        time = vm.collectForm.collectInterval+':00',
                        params = {
                            deviceCode: row.small_cell_code,
                            serialNumber: row.serial_number,
                            type: 'enb',
                            operatorCode: operatorCodeGloab,
                            collectInterval: vm.collectForm.collectInterval
                        },
                        paramsExist = {
                            type: 'enb',
                            operatorCode: operatorCodeGloab
                        };

                    axios.post('${ctx}/trace/isExistTracingDevice.action', stringify(paramsExist)).then(function(res){
                        var data = res.data;

                        if(data && data.isExist == 'true') {
							vm.$message.error('SN=' + data.serialNumber + '<%=rb.getString("ZhengZaiShouJi")%>');
                        }else {
                            axios.post('${ctx}/trace/start.action', stringify(params)).then(function(res){
                                var data = res.data;

                                if(data.success == true) {
                                    queryVue.startInterval(time);
                                    queryVue.queryLatestInfo();
                                    vm.collectMessageShow = false;
                                    
                                    vm.$message.success('<%=rb.getString("ChengGong")%>');
                                }else {
                                    vm.$message.error(data.message);
                                }
                            });
						}
                    });
                },

            	parseMME(str) {
                    if(str) {
            		    return eval('('+str+')');
                    }else {
                        return [];
                    }
            	},
            	parsePLMN(row,value,index) {
            		var vm = this, stringOrObj = vm.plmnIsJson(value);
            		if(stringOrObj == true){
            			//对象
            			var curValue = JSON.parse(value),
                            length = 0;

                        Object.keys(curValue).map(function(code){
                            if(curValue[code]) length += curValue[code].split(',').length;
                        });

            			return length;
            		}
            	},
            	parseStrPLMN(row,value,index) {
            		var vm = this, html = '', stringOrObj = vm.plmnIsJson(value);
            		//字符串
            		if(stringOrObj == false){
            			var list = (value||'').split(','),
            			html = list[0];

	                    if(list.length>1) {
	                    	html += '<a style="color: blue;" onclick="showPopLayer({event: event, html: &quot;<di style=\'padding-top: 15px;display: block;\'>'+value+'</div>&quot;})">[' + list.length
	                                +'<i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;"></i>]</a>';
	                    }
	                    return html;
            		}
            	},
             	plmnIsJson(str){
                	if(typeof(str) == 'string'){
                		try{
                			var curValue = JSON.parse(str);
                			//对象
                        	if(typeof curValue == 'object' && curValue){
                        		return true;
                        	}else{
                        		//字符串
                        		return false;
                        	}
                		}catch(e){
                			return false;
                		}
                	}else{
                		return false;
                	}
                },
                refreshList() {
                    var vm = this;
                    // 兼容旧的el-ctable和新的vxe-table
                    if (vm.$refs.list && vm.$refs.list.refresh) {
                        vm.$refs.list.refresh();
                    } else {
                        // vxe-table数据刷新
                        vm.loadTableData();
                    }
                },
                // vxe-table数据加载方法
                loadTableData() {
                    var vm = this;
                    vm.tableLoading = true;
                    
                    /* 原始数据加载逻辑 - 暂时注释 */
                    axios.post(vm.tbURL, stringify(vm.queryParams)).then(function(response) {
                        var data = response.data;
                        if (data && data.rows) {
                            vm.tableData = data.rows;
                            vm.pager.total = data.total || 0;
                            vm.updatePageData();
                        } else if (Array.isArray(data)) {
                            vm.tableData = data;
                            vm.pager.total = data.length;
                            vm.updatePageData();
                        }
                        vm.tableLoading = false;
                        vm.loadSuccess(data);
                    }).catch(function(error) {
                        vm.tableLoading = false;
                    });
                },
                // 分页数据更新
                updatePageData() {
                    var vm = this;
                    
                    Object.assign(vm.queryParams, {
                        page: vm.pager.currentPage,
                        rows: vm.pager.pageSize
                    });
                },
                sizeChange(size){
                    var vm = this;
                    vm.pager.pageSize = size;
                    vm.updatePageData();
                    // 滚动到顶部
                    if (vm.$refs.xTable) {
                        vm.$refs.xTable.scrollTo(0, 0);
                    }
                },
                currentChage(page){
                    var vm = this;
                    vm.pager.currentPage = page;
                    vm.updatePageData();
                    // 滚动到顶部
                    if (vm.$refs.xTable) {
                        vm.$refs.xTable.scrollTo(0, 0);
                    }
                },
                loadSuccess(data) {
                    $('#cellInfo').css('width','99.9%');
					setTimeout(function(){
						$('#cellInfo').css('width','100%');
					},2000);
                	refresh_cellStatusStatistics();
                },
                showExport() {
                    /*
                    $('.export-content').html('');
                    $('.export-content').each(function(idx,item){
                        if($(item).is(':visible')) {
                            $(item).load('${ctx}/cell/cpeinfos/toExportConfig.action',function(html) {
                            })
                        }
                    })
                    */
                    $('#enb_export_content').load('${ctx}/cell/cpeinfos/toExportConfig.action',function(html) { });
                },
                showAddOrImport() {
                    
                    $('.addOrImport-content').html('');
                    $('.addOrImport-content').each(function(idx,item){
                        if($(item).is(':visible')) {
                            $(item).load('${ctx}/cell/cpeinfos/addOrImportDevice.action',function(html) {
                               
                            })
                        }
                    })
                },
                // 获取所有列
                getAllDefaultCols() {
                    var vm = this,
                        columns = [
                            {label: '<%=rb.getString("GaoJingShu")%>', prop: 'alarm', width: 65},
                            {label: '<%=rb.getString("XiaoZhanBianMa")%>', prop: 'serial_number', width: 180},
                            {label: '<%=rb.getString("HostName")%>', prop: 'host_name', width: 150, formatter: vm.nameFmt},
                            {label: '<%=rb.getString("ShePinKaiGuanZhuangTai")%>', prop: 'rf_status', width: 95},
                            {label: '<%=rb.getString("ShiFouJiHuo")%>', prop: 'op_state', width: 120},
                            {label: 'ECI', prop: 'CELL_IDENTITY', width: 70},
                            {label: '<%=rb.getString("PCI2")%>', prop: 'PHYCELLID', width: 50},
                            {label: '<%=rb.getString("MMEZhuangTai")%>', prop: 'mme_status', width: 80},
                            {label: '<%=rb.getString("UEShu")%>', prop: 'ue_count', width: 80},
                            {label: '<%=rb.getString("EUShu")%>', prop: 'euCountStr', width: 80},
                            {label: '<%=rb.getString("RUShu")%>', prop: 'ruCountStr', width: 80},
                            {label: '<%=rb.getString("CPELianJieShu")%>', prop: 'cpe_connect', width: 80},
                            {label: '<%=rb.getString("IPDiZhi")%>', prop: 'cell_ip', width: 100},
                            {label: '<%=rb.getString("MACDiZhi")%>', prop: 'mac_address', width: 115}
                        ];

                    columns.push({label: '<%=rb.getString("EnodebId")%>', prop: 'enbId', width: 100});
                    columns.push({label: '<%=rb.getString("XIAOQUID")%>', prop: 'cellId', width: 100});

                    columns.push({label: '<%=rb.getString("ChanPinLeiXingBiaoZhi")%>', prop: 'product', width: 110});
                    columns.push({label: '<%=rb.getString("ChanPinMingCheng")%>', prop: 'product_name', width: 110});

                    columns.push({label: '<%=rb.getString("SheBeiXingHaoMing")%>', prop: 'module_type', width: 95});
                    columns.push({label: '<%=rb.getString("SoftwareVersion")%>', prop: 'software_version', width: 120});
                    columns.push({label: '<%=rb.getString("SheBeiZu")%>', prop: 'group_name', width: 130});

                    if("${isSuperAdmin}" == '1'){
                        columns.push({label: '<%=rb.getString("IPSECDiZhi")%>', prop: 'IPSEC_ADDR', width: 100});
                        columns.push({label: '<%=rb.getString("MMEPoolIPSECDiZhi")%>', prop: 'mmepool_ipsec_addr', width: 100});
                    }
                    /* 目前只有Amara支持ups，后面后端会调整逻辑 */
                    /* if(siteIdShow == 'true'){
                        columns.push({label: siteIdLabel, prop: 'site_id', width: 70});
                    }*/
                    
                   //以下字段受此参数('${enbAdditionalColShow}')控制显示或隐藏
                   if(supportTopoSite || vm.enbAdditionalColShow == 'true') {
						columns.push({label: siteNameLabel, prop: 'sub_station_name', width: 120});
                   }
                   if(vm.enbAdditionalColShow == 'true' || registerSiteIdShow  == 'true'){
                        columns.push({label: siteIdLabel, prop: 'site_id', width: 80});
                   }
				   if(vm.enbAdditionalColShow == 'true'){
	                    columns.push({label: '<%=rb.getString("AnZhuangXiangXiDiZhi")%>', prop: 'install_address', width: 200});
	                    columns.push({label: '<%=rb.getString("ZhuangTai")%>', prop: 'service_status', width: 120});
	                    columns.push({label: 'Rom', prop: 'rom', width: 120});
					}
                    
                    columns.push({label: '<%=rb.getString("PinDian")%>', prop: 'EARFCNDLINUSE', width: 120});
                    columns.push({label: '<%=rb.getString("TongBuZhuangTai")%>', prop: 'synStatus', width: 120});
                    columns.push({label: '<%=rb.getString("KPIShangBaoZhuangTai")%>', prop: 'pm_report_status', width: 130});
                    columns.push({label: '<%=rb.getString("GPSWeiXingShu")%>', prop: 'gps_satellite_count', width: 75});
                    columns.push({label: '<%=rb.getString("YunXingShiJian")%>', prop: 'up_time', width: 110});
                    
                    columns.push({label: '<%=rb.getString("LeiJiShiChang")%>', prop: 'online_duration', width: 110});
                    columns.push({label: '<%=rb.getString("DiYiCiLianJieShiJian")%>', prop: 'first_online_time', width: 125});
                    columns.push({label: '<%=rb.getString("ShangCiLianJieShiJian")%>', prop: 'LASTINFORMTIME', width: 125});
                    
                    columns.push({label: '<%=rb.getString("JieRuShiJian")%>', prop: 'online_time', width: 125});
                    columns.push({label: '<%=rb.getString("DuanKaiShiJian")%>', prop: 'offline_time', width: 125});

                    columns.push({label: '<%=rb.getString("JiZhanZhiShi")%>', prop: 'network_model', width: 90});
                    columns.push({label: '<%=rb.getString("FirmwareVersion")%>', prop: 'firmware_version', width: 115});
                    columns.push({label: '<%=rb.getString("GPSBanBen")%>', prop: 'gps_version', width: 90});

                    if(isCloud == 'true' && false) {
                        columns.push({label: '<%=rb.getString("XiaoQuKeYongZhanBi")%>', prop: 'available_rate', width: 70});
                    }
                    //cloud版的支持halob 
                    if( isSupportHalob == 'true') {
                        columns.push({label: 'HaloX', prop: 'halob_flag', width: 100});
                    }

                    columns.push({label: '<%=rb.getString("PLMN")%>', prop: 'plmnid', width: 60});
                    columns.push({label: '<%=rb.getString("DaiKuan")%>', prop: 'bandwidth', width: 60});
                    columns.push({label: '<%=rb.getString("TAC")%>', prop: 'tac', width: 50});
                    columns.push({label: '<%=rb.getString("ZiZhenPeiBi")%>', prop: 'signment', width: 50});
                    columns.push({label: '<%=rb.getString("TeShuZiZhenPeiBi")%>', prop: 'specialSubframe', width: 50});
                    columns.push({label: '<%=rb.getString("GenXuLieSuoYin")%>', prop: 'rootIndex', width: 50});
                    columns.push({label: '<%=rb.getString("GPSJingDu")%>', prop: 'gps_longitude', width: 80});
                    columns.push({label: '<%=rb.getString("GPSWeiDu")%>', prop: 'gps_latitude', width: 70});
                    columns.push({label: '<%=rb.getString("GPSGaoDu")%>', prop: 'gps_height', width: 70});
                    
                    columns.push({label: 'Mechanical Downtilt', prop: 'mechanical_downtilt', width: 135});
                    columns.push({label: 'Electronic Downtilt', prop: 'electronic_downtilt', width: 135});
                    columns.push({label: 'Vertical 3dB beam width', prop: 'vertical_3dB_beam_width', width: 160});
                    columns.push({label: 'Horizontal Azimuth', prop: 'horizontal_azimuth', width: 135});

                    if(writableMap["CODE_ENB_EXPIRY_DATE"] != undefined) {
                        columns.push({label: '<%=rb.getString("YouXiaoQi")%>', prop: 'validity', width: 125});                        
                    }
                    columns.push({label: '<%=rb.getString("SuoDingZhuangTai")%>', prop: 'lock_status', width: 90});
                    columns.push({label: '<%=rb.getString("CPETxPower")%>', prop: 'tx_power', width: 90});
                    columns.push({label: '<%=rb.getString("WanZhuangTai")%>', prop: 'wanSpeed', width: 90});
                    columns.push({label: vm.currentRemarkLabel, prop: 'remark', width: 125});
                    // columns.push({label: '<%=rb.getString("JianQuanMa")%>', prop: 'authCode', width: 120});
                    return columns;
                },
                selectionChange(s) {
                    this.selectedRows = s||[];
                },
                // vxe-table checkbox选择变更事件处理
                handleSelectionChange({ records }) {
                    var vm = this;
                    vm.selectedRows = records || [];
                },
                // vxe-table 排序变更事件处理
                handleSortChange({ field, order }) {
                    var vm = this;
                    // 将排序参数映射到 queryParams
                    vm.queryParams.sort = field || '';
                    // vxe-table order: 'asc'/'desc'/null, 后端需要: 'asc'/'desc'/''
                    vm.queryParams.order = order || '';
                    // 排序时重置到第一页
                    vm.pager.currentPage = 1;
                    vm.queryParams.page = 1;
                },
                closeSyncName() {
                    document.body.click();
                },
                clearSelection() {
                    var vm = this;
                    
                    if (vm.$refs.xTable) {
                        vm.$refs.xTable.clearCheckboxRow();
                    }
                    vm.selectedRows = [];
                },         
                movecells(){
                	var vm = this;
                    if(vm.selectedRows.length == 0)return
                	vm.groupId = '';
                	vm.deviceGroupTip = false;
                	vm.deviceGroupMoveShow = true;
                },
                //选择设备选中事件
    	    	groupIdChange(row, old){	    		
    				var vm = this;
    				if(row) {
    					vm.groupId = row.id;	
    					vm.deviceGroupTip = false;
    				}			
    			},
                deviceGroupMoveSubmit(){
                	var vm = this, params = {},
                		cellCodes = vm.selectedRows.map(function(item){ return item.small_cell_code }).join(',');
                	
	                params.toGroupId = vm.groupId;
	                params.ids = cellCodes;
                	//根据被选中的ca列表id进行提示
    	    		if(vm.groupId == '' || vm.groupId == null || vm.groupId == undefined){
    					vm.deviceGroupTip = true;
    					return false;
    				}else{
    					vm.deviceGroupTip = false;	    					
    				}
    	    		axios.post('${ctx}/system/deviceGroup/moveCellToGroup.action',stringify(params)).then(function(response){
    					let data = response.data;
    					if(data.success){
    						vm.$message({
    							message: '<%=rb.getString("ChengGong")%>',
    							type:'success'
    						});	
    						vm.deviceGroupMoveShow = false;
    						vm.clearSelection();
            				vm.refreshList();
    					}else{
    						vm.$message({
    							type:'error',
    							message:data.message
    						})
    					}
    					
    				}).catch(function(error){}) 
                },
                syncList() {
                	var vm = this,
                		codes = vm.selectedRows.map(function(item){ return item.serial_number}),
                		rows = vm.tableData,
            			cellCodes = '';

                	if(vm.selectedRows.length == 0)return

                	/*rows.map(function(row){
                		if(codes.includes(row.serial_number)) {
                			if(row.connection_status != 'Off'){
                				row.connection_status = 'updating';
                			}
                		}
                	});*/
                	
                	cellCodes = vm.selectedRows.map(function(item){
            			return item.small_cell_code
            		}).join(',');
                	
            		var params = {
            			cellCodes : cellCodes
            		}
                    vm.batchSync = true;
                    vm.batchCode = cellCodes;
                    openSyncDialog();
            		
                },
                closeSyncDialog(){
                	var vm = this;
                	vm.batchSync = false;
                	vm.batchCode = '';
                    vm.clearSelection();
                	
                },
                rebootList() {
                	var vm = this,
                		checkedRow = vm.selectedRows,
                		cellCodes = '';
                	if(vm.selectedRows.length == 0)return
            		checkedRow.map(function(item){
            			cellCodes += item.small_cell_code + ',';
            		});
            		
            		$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingChongQiSheBei")%>", function (r) {
            			if (r) {
            				showMsg('prompt_msg',"<%=rb.getString("MingLingYiXiaFa")%>")
            				var param = {
            					cellCodes: cellCodes
            				};
            				$.post("${ctx}/task/reboot/batchRebootCell.action", param, function (data) {
            					if (data["success"]) {
            						vm.clearSelection();
            					}else{
            						showMsg('error_msg',data["message"]);
            					}
            				}, "json");
            			}
            		}).addClass("seriousConfirm");
                },
                // 批量移入回收站设备
                recycleCells(){
                    var vm = this,
                        params = {},
                        urls= '${ctx}/recycle/moveDeviceToRecycle.action'
                        idsList = [];
                    if(vm.selectedRows.length <= 0)return
                    vm.selectedRows.map((item,index) => {
                        idsList.push(item.small_cell_code);
                    })
                    params.smallCellCodeStr = idsList.join(',');
                    var confirmHint ='<div style="font-size:14px;color:#333333">'+ '<%=rb.getString("QueRenJiangSheBeiYiRuHuiShouZhan")%>' +'</div>'+'<div style="font-size:12px;color:#999999">'+ '<%=rb.getString("YiRuHuiShouZhanTiShi")%>' +'</div>';
                    vm.$confirm(confirmHint,'<%=rb.getString("QueRen")%>',{
                        customClass:"warningConfirm",
                        confirmButtonText:'<%=rb.getString("QueDing")%>',
                        cancalButtonText:'<%=rb.getString("QuXiao")%>',
                        type:'warning',
                        dangerouslyUseHTMLString:true
                    }).then(()=>{
                        axios.post(urls,stringify(params)).then(function(response){
                            let data = response.data;
                            if ( data.success ){
                                vm.$message({
                                    message: '<%=rb.getString("ChengGong")%>' ,
                                    type:'success',
                                })
                                vm.loadTableData();
                                vm.selectedRows = [];
                                if (vm.$refs.xTable) {
                                    vm.$refs.xTable.clearCheckboxRow();
                                }
                            }else {
                                vm.$message.error(data.message)
                            }
                        }).catch(function(error){})
                    }).catch(()=>{})
                },
                syncName(code, newName) {
                    var vm = this;

                    $.post('${ctx}/cell/cpeinfos/syncCellName.action?smallCellCode='+code, function(data){
                        if(data.success){
            				vm.refreshList();
                            vm.closeSyncName();
                        }else{
                            showMsg('error_msg',data["message"]);
                        }
                    },'json'); 
                },
                synchronizeGPS(code) {
                    var vm = this;

                    $.ajax({
                        url: '${ctx}/cell/topo/syncGPSInfo.action',
                        type: 'post',
                        data: {cell_code: code},
                        dataType: 'json',
                        success: function(data) {
                            if(data.success){
                                vm.loadTableData();
                            }else{
                                showMsg('error_msg',data["message"])
                            }
                        }
                    });
                },
                //锁定图标点击
                lockUnlockBtnClick(row){
                	var vm = this;
                	vm.curRowSn = row.small_cell_code;
                	//lock_operation: 0-关闭锁定MAC; 1-自动绑定MAC
                	if(row.lock_operation === '' || row.lock_operation === null || row.lock_operation === undefined){
                		vm.lockForm.lockStatus = '0';
                	}else{
                		vm.lockForm.lockStatus = row.lock_operation;
                	}
                	vm.addLoading = false;
                	if(row){
                        if(row.lock_type == '0'){
                        	if(row.lock_status == '0'){
                        		//解锁不显示 锁定原因，锁定显示
                        		vm.lockReasonShow = false;
                        	}else{
                        		vm.lockReasonShow = true;
                        		vm.lockReason = row.lock_reason;
                        	}
                        	vm.lockMac = row.lock_mac;
                        	vm.lockMessageShow = true;
                        }
                    }               	
                },
                lockSubmit(){
                	var vm = this, curLockStatus = '';
                	if(vm.lockForm.lockStatus == '0'){
                		curLockStatus = 'RESERVED';
                	}else{
                		curLockStatus = 'AUTO';
                	}
                	vm.addLoading = true;
               		axios.post('${ctx}/cell/cpeinfos/cellModifyLockStatus.action',stringify({small_cell_code: vm.curRowSn, lockStatus: curLockStatus})).then(function(response){
               			var data = response.data;
               			if(data){
               				vm.addLoading = false;
               				if(data["success"]){
               					vm.$message({
            						message:"<%=rb.getString("ChengGong")%>",
            						type:'success'
            					});
               					vm.lockMessageShow = false;
               					vm.refreshList();
               				}else{
               					vm.$message.error(data["message"])
               				} 
               			}
           			}) 
                },
                // table formatters
                connStatusFmt(row, value, index) {

                    return connStatusFormatterSyn(value, row, index);
                },
                alarmToInfo(row){
                    var vm = this;
                    //goCellDetailAlarmInfoWin(row)
                    vm.openSettingPage(row,'alarm');
                },
                rfStatusFmt(row, value, index) {
                    return RFStatusFormatter(value, row, index);
                },
                activeMultFmt(row, value, index) {
                    var html = '',
                        states = (value+'').split(',');

                    if(states.length>1) {
                        states.map(function(state, idx){
                            if(state == '1') {
                                html += ['<div class="activeStatusItem">',
                                            '<span class="el-icon el-icon-status-active"></span>',
                                            '<span class="status-tip">Cell'+(idx+1)+' Status: <%= rb.getString("JiHuo")%></span>',
                                        '</div>'].join(' ');
                            }else {
                                html += ['<div class="inactiveStatusItem">',
                                            '<span class="el-icon el-icon-status-active"></span>',
                                            '<span class="status-tip">Cell'+(idx+1)+' Status: <%= rb.getString("QuJiHuo")%></span>',
                                        '</div>'].join(' ');
                            }
                        });
                    }else {
                        if(states[0] == '1') {
                            html += ['<div class="activeStatusItem">',
                                            '<span class="el-icon el-icon-status-active"></span>',
                                            '<span class="status-tip">Cell1 Status: <%= rb.getString("JiHuo")%></span>',
                                        '</div>',
                                        '<div class="activeStatusItem">',
                                            '<span class="el-icon el-icon-status-active"></span>',
                                            '<span class="status-tip">Cell2 Status: <%= rb.getString("JiHuo")%></span>',
                                        '</div>'].join(' ');
                        }else {
                            html += ['<div class="inactiveStatusItem">',
                                        '<span class="el-icon el-icon-status-active"></span>',
                                        '<span class="status-tip">Cell1 Status: <%= rb.getString("QuJiHuo")%></span>',
                                    '</div>',
                                    '<div class="inactiveStatusItem">',
                                        '<span class="el-icon el-icon-status-active"></span>',
                                        '<span class="status-tip">Cell2 Status: <%= rb.getString("QuJiHuo")%></span>',
                                    '</div>'].join(' ');
                        }
                    }

                    return html;
                },
                oldMmeStatusFmt(rowData,value,type){
                    var vm = this,
                        mmeStatus,textVal,value,hasDisconn=false,hasConn=false,mmeDataList=[],leftStr='',parseMMEOnNum=0;
                
                    if (value == null || value == "") {
                        mmeStatus ='';
                        textVal = '--'
                    }else{
                        if (value == "1") {
                            mmeDataList.push({mmeIp:rowData.s1siglinkserverlist,status:'1'})
                            mmeStatus ='';
                            textVal = '<%= rb.getString("MMEYiLianJie")%>'
                        } else if (value == "0") {
                            mmeDataList.push({mmeIp:rowData.s1siglinkserverlist,status:'0'})
                            mmeStatus ='offStatusCls';
                            textVal = '<%= rb.getString("MMEWeiLianJie")%>'
                        } else if (value == "2") {
                            mmeStatus ='';
                            textVal = '--'
                        } else{
                            var mmePools = value.split(",");
                            for(var i = 0; i < mmePools.length; i++ ){
                                var mme = mmePools[i];
                                var mmeArr = mme.split("=");
                                
                                if(mmeArr.length == 2 && "mme1" == mmeArr[0]){
                                    mmeDataList.push({mmeIp:rowData.mme_pool_1,status:mmeArr[1]})
                                }else if(mmeArr.length == 2 && "mme2" == mmeArr[0]){
                                    mmeDataList.push({mmeIp:rowData.mme_pool_2,status:mmeArr[1]})
                                }
                            }
                            mmeDataList.map((item)=>{
                                if (item.status == '1'){
                                    hasConn = true;
                                }else if(item.status == '0'){
                                    hasDisconn = true
                                }
                            })
                            if (hasDisconn == true){
                                if (hasConn == false) {
                                    mmeStatus ='offStatusCls';
                                    textVal = '<%= rb.getString("MMEWeiLianJie")%>'
                                }else{
                                    mmeStatus ='offOrOnStatusCls';
                                    textVal = '<%= rb.getString("MMEYiLianJie")%>'
                                }
                            }else {
                                mmeStatus ='';
                                textVal = '<%= rb.getString("MMEYiLianJie")%>'
                            }
                        }
                    }
                    mmeDataList.map((item)=>{
                        if (item.status == '1'){
                            parseMMEOnNum += 1;
                        }
                    })
                    if(type == 'left'){
                        value = "<span class='"+ mmeStatus +"'>" + textVal +"</span>"
                    }else if(type == 'right'){
                        value = parseMMEOnNum + '/' + mmeDataList.length
                    }else if(type == 'list'){
                        value = mmeDataList
                    }
                    return value;
                },
                mmesStatusFmt(str){
                	var mmelist = eval('('+str+')');
                	var mmeStatus,textVal,value,hasDisconn=false,hasConn=false;
                	mmelist.map(function(item){
                		if (item.status == '1'){
                			hasConn = true;
                		}else if(item.status == '0'){
                			hasDisconn = true
                		}
                	})
                	
                	if (hasDisconn == true){
                		if (hasConn == false) {
                			mmeStatus ='offStatusCls';
                            textVal = '<%= rb.getString("MMEWeiLianJie")%>'
                		}else{
                			mmeStatus ='offOrOnStatusCls';
                            textVal = '<%= rb.getString("MMEYiLianJie")%>'
                		}
                	}else {
                		mmeStatus ='';
                        textVal = '<%= rb.getString("MMEYiLianJie")%>'
                	}
                	
                	value = "<span class='"+ mmeStatus +"'>" + textVal +"</span>"
                	return value;
                },
                plmnFmt(row, value, index) {
                    var vm = this,
                        list = (value||'').split(','),
                        str = list[0];

                    if(list.length>1) {
                        str += '<a style="color: blue;" onclick="showPopLayer({event: event, html: &quot;<di style=\'padding-top: 15px;display: block;\'>'+value+'</div>&quot;})">[' + list.length
                                +'<i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;"></i>]</a>';
                    }

                    return str;
                },
                plmnSpecialFmt(row, value, index) {
                    if (value === null || value === "" || value === undefined) {
		                return null;
		            }
	            },
                plmnNewFmt(row, value, index) {  
                    var vm = this, html ='', plmnArr = [];
                    
                    if(value === '' || value === null || value === undefined){
                    	return '';
                    	
                    }else{
                    	var curValue = JSON.parse(value);
                    	if(curValue){
            				var curKeyList = Object.keys(curValue);
	                		if(curKeyList.length == 3){
	                			html = "<div class='mme-list-item objectPlmnItem'>" + Object.keys(curValue)[0] + " : " + Object.values(curValue)[0] + "</div>"+ 
				            			"<div class='mme-list-item objectPlmnItem'>" + Object.keys(curValue)[1] + " : " + Object.values(curValue)[1] + "</div>"+ 
				            			"<div class='mme-list-item objectPlmnItem'>" + Object.keys(curValue)[2] + " : " + Object.values(curValue)[2] + "</div>";
	                		}else if(curKeyList.length == 2){
	    	                	html = "<div class='mme-list-item objectPlmnItem'>" + Object.keys(curValue)[0] + " : " + Object.values(curValue)[0] + "</div>"+ 
				            		   "<div class='mme-list-item objectPlmnItem'>" + Object.keys(curValue)[1] + " : " + Object.values(curValue)[1] + "</div>";
	    	                }else{      	                	
	    	                	html = "<div class='mme-list-item objectPlmnItem'>" + Object.keys(curValue)[0] + " : " + Object.values(curValue)[0] + "</div>";
	    	                }
				            return html;
                    	}
                        
                    }
                },
                lteOrGsmplmnNewFmt(row, value, index) {  
                    var vm = this, html ='', plmnArr = [];
                    
                    if(value === '' || value === null || value === undefined){
                    	return '';
                    	
                    }else{
                    	var curValue = JSON.parse(value);
                    	if(curValue){
            				var curKeyList = Object.keys(curValue);
	                		if(curKeyList.length == 3){
	                			html = "<div style='height:30px; font-size: 12px; padding:0 10px;border: 1px solid #E9E9E9;display:flex;align-items: center;margin-left:10px;'>" + 'LTE Cell 1' + " : " + Object.values(curValue)[0] + "</div>"+ 
				            			"<div style='height:30px; font-size: 12px; padding:0 10px;border: 1px solid #E9E9E9;display:flex;align-items: center;margin-left:10px;'>" + 'LTE Cell 2' + " : " + Object.values(curValue)[1] + "</div>"+ 
				            			"<div style='height:30px; font-size: 12px; padding:0 10px;border: 1px solid #E9E9E9;display:flex;align-items: center;margin-left:10px;'>" + 'LTE Cell 3' + " : " + Object.values(curValue)[2] + "</div>";
	                		}else if(curKeyList.length == 2){
	    	                	html = "<div style='height:30px; font-size: 12px; padding:0 10px;border: 1px solid #E9E9E9;display:flex;align-items: center;margin-left:10px;'>" + 'LTE Cell 1' + " : " + Object.values(curValue)[0] + "</div>"+ 
				            		   "<div style='height:30px; font-size: 12px; padding:0 10px;border: 1px solid #E9E9E9;display:flex;align-items: center;margin-left:10px;'>" + 'LTE Cell 2' + " : " + Object.values(curValue)[1] + "</div>";
	    	                }else{      	                	
	    	                	html = "<div style='height:30px; font-size: 12px; padding:0 10px;border: 1px solid #E9E9E9;display:flex;align-items: center;margin-left:10px;'>" + 'LTE Cell 1' + " : " + Object.values(curValue)[0] + "</div>";
	    	                }
				            return html;
                    	}
                        
                    }
                },
                ueCountFmt(row, value, index) {
                    return ueCountsFormatter(value, row, index);
                },
                euruCountFmt(row, value, index) {
                    return euruCountsFormatter(value, row, index);
                },
                cpeCountFmt(row, value, index) {
                    return cpeCountsFormatter(value, row, index);
                },
                ipAddrFmt(row, value, index) {
                    if(value) {
                        value = '<a href="https://' + value + '" target="_blank" style="color: #4d84ff;">'+value+'</a>';
                    }

                    return value;
                },
                productFmt(row, value, index) {
                    if ("${carrierType}" == "1") {
                        return value;
                    }else {
                        if(row.have_connected == 2) return '';
                        if(value == '--') return '--';

                        return value;
                    }
                },
                capablityFmt(rowData,value,rowIndex){
                    var rowDatas = rowData,
                        rowIndexs = rowIndex,
                        capablity = rowData.capablity;

                    if(capablity == 'enable' && isLWAEnable){
                        value = "<span class='cpeLwaKai' style='font-size:22px'>"+ "<span style='font-size:12px;margin-left:25px'>"+(value)+"</span>"+"</span>"
                    }else if(capablity =='disable' && isLWAEnable){
                        value = "<span class='cpeLwaWu' style='font-size:22px'>"+ "<span style='font-size:12px;margin-left:25px'>"+(value)+"</span>"+"</span>"
                    }
                    return value;
                },
                earfcnFmt(row, value, index) {
                    if(value) {
                        var resList = [];
                        value.split(',').map(function(item){
                            resList.push(earfcnFormatter(item, row, index));
                        });
                        
                        return resList.join(',');
                    }else {
                        return '';
                    }
                },
                syncStatusFmt(rowData, value, rowIndex){
                    var val = '';
                    if (value == null) {
                        return null;
                    } else if (value == "--" ){
                        return "--";
                    } else if (value == ("GPS " +"<%= rb.getString("ZhengZaiTongBu")%>")) {
                        val ="GPS "+ "<%= rb.getString("ZhengZaiTongBu")%>";
                        value = "<span>"+(val)+"</span>"
                    }else if (value == ("1588 " +"<%= rb.getString("ZhengZaiTongBu")%>")) {
                        val ="1588 "+ "<%= rb.getString("ZhengZaiTongBu")%>";
                        value = "<span>"+(val)+"</span>"
                    }else if (value == ("REM " +"<%= rb.getString("ZhengZaiTongBu")%>")) {
                        val ="REM "+ "<%= rb.getString("ZhengZaiTongBu")%>";
                        value = "<span>"+(val)+"</span>"
                    }else if (value == "GPS "+ "<%= rb.getString("TongBuChengGong")%>" ) {
                        val ="GPS "+ "<%= rb.getString("TongBuChengGong")%>";
                        value = "<span>"+(val)+"</span>"
                    }else if (value == "1588 "+"<%= rb.getString("TongBuChengGong")%>" ) {	
                        val = "<%= rb.getString("TongBuChengGong")%>";
                        val = "1588 " + val;
                        value = "<span>"+(val)+"</span>"
                    }else if (value == "REM "+"<%= rb.getString("TongBuChengGong")%>") {	
                        val = "<%= rb.getString("TongBuChengGong")%>";
                        val = "REM " + val;
                        value = "<span>"+(val)+"</span>"
                    }else if (value == "<%= rb.getString("WeiTongBu")%>") {	
                        val = "<%= rb.getString("WeiTongBu")%>";
                        value = "<span class='offStatusCls'>"+(val)+"</span>"
                    }
                    return '<div style="min-width: 130px;diplay: line-block;">'+value+'</div>';
                },
                kpiStatusFmt(row, value, index){
                	var zhengchang = '<%= rb.getString("ZhengChang")%>',
                		guanbi = '<%= rb.getString("Guan")%>',
                		yichang = '<%= rb.getString("SunHuai")%>';
                    if(value == null){
                        return null;
                    }else if(value == "off"){
                        value = "<span>"+(guanbi)+"</span>"		
                    }else if(value == "normal"){
                        value = "<span>"+(zhengchang)+"</span>"
                    }else if(value == "broken"){
                        value = "<span class='offStatusCls'>"+(yichang)+"</span>"
                    }
                    return value;
                },
                halobStatusFmt(rowData, value, rowIndex) {
                    if ("1" == value) {
                        value = "<span class='el-icon el-icon-status-enable' style='margin-right:5px;font-size: 20px;'></span>On";
                    } else if ("0" == value) {
                        value = "<span class='el-icon el-icon-status-disable' style='margin-right:5px;font-size: 20px;'></span>Off";
                    } else {
                        value = "--";
                    }
                    return value;
                },
                validityFmt(rowData, value, rowIndex) {
                    if(value == '--'){
                        return value;
                    }else{
                        if(rowData.lock_status == 0){
                            return "<span>"+value+"</span>"
                        }else if(rowData.lock_status == 1){
                            return "<span style='color:#D2D2D2'>"+value+"</span>"
                        }else{
                            return ''
                        }
                    }
                },
                /*lockStatusFmt(rowData, value, rowIndex) {
                	
                	if(value == "--"){
                        return value;
                    }else{
                        if(value == 0){
                            return "<div style='display: flex; align-items: center;'><span class='el-icon el-icon-status-unlock' style='margin-right:5px;font-size:20px;'></span><%=rb.getString("YouXiaoQiJieSuo")%></div>";
                        }else{
                            return "<div style='display: flex; align-items: center;'><span class='el-icon el-icon-operation-lock' style='margin-right:5px;font-size:20px;'></span><%=rb.getString("YouXiaoQiSuoDing")%></div>";
                        }
                    }
                },*/
                getEnbGPSSignalData(rowData) {
                    var vm = this,
                        enb_code = rowData.small_cell_code,
                        url = "${ctx}/cell/param/getSatellitesDataList.action?timeZone="+timeZone+'&smallCellCode='+enb_code+'&rd='+Math.random();
                    
                    vm.satelliteURL = url;
                    vm.sateloading = true;
                    vm.$refs.satelite.showSlide(function(){
                        setTimeout(function(){
                            vm.sateloading = false;
                        },1000);
                    });
                },
                closeSatellite() {
                    this.$refs.satelite.hide();
                },
                // table sliders
                closeUeSlide() {
                    this.$refs.ueCount.hide();
                },
                closeCpeSlide() {
                    this.$refs.cpeCount.hide();
                },
                closeActive() {
                    this.$refs.activeRatio.hide();
                },
                getUnuseTime(row) {
                    getEnbUnuseTimeData(row);
                },

                tabClick(tab) {
                    if(tab.name == 'map') {
                        loadEnbTopo();
                    }
                },
                openSettingPage(row,page){
                    this.selectedRow = row;
                    enbPlatform = row.platform_flag;
                    enbPlatformType = row.platformType;

                    // 检查 eNBTopo_tab.jsp 是否已打开 setting.jsp 页面，如果打开则先关闭以防止 ID 冲突
                    if(typeof topovm !== 'undefined') {
                        try {
                            // 直接尝试关闭 eNBTopo_tab 的 slide，即使它没有打开也不会报错
                            topovm.$refs.topoSettingSlide.hide();
                        } catch(e) {}
                    }
                	this.settingUrl = '${ctx}/enb/setting/openSettingPage.action';
                	this.$refs.settingPage.showSlide(function(){
                		eventBus.$emit("row-data",row,page,'monitor')
                    });
                },
				closeSettingPage(){
                	this.$refs.settingPage.hide();
                    this.loadTableData();
                },

                optClick(row, evt) {
                    var vm = this,
                        rowDatas = row,
                        status = rowDatas.connection_status,
                        idval = rowDatas.small_cell_code+"",
                        flag = rowDatas.platform_flag,
                        isCA = rowDatas.ca_flag,
                        opState = rowDatas.op_state,
                        gsmOpState = rowDatas.gsm_cells_op_state,
                        ip = rowDatas.cell_ip,
                        rfStatus = rowDatas.rf_status||'',
                        //lockStatus = rowDatas.lock_status,
                        validitySwitch = rowDatas.validity_switch,
                        CELL_IDENTITY = rowDatas.CELL_IDENTITY,
                        product = rowDatas.product;
                        serialNumber = rowDatas.serial_number,
                        dualCarrierType = rowDatas.dual_carrier_type,
                        collectShow = dualCarrierType != 2;

                    vm.selectedRow = row;
                    
                    var connectStatus = status;
                    var deviceType = flag; // intel -- 0 ,gaotong -- 1
                    var XinXi = '<%=rb.getString("XinXi")%>';
                    var TongBu = '<%=rb.getString("TongBu")%>';
                    var SheZhi = '<%=rb.getString("SheZhi")%>';
                    var MeiYouQuanXian = "<%=rb.getString("MeiYouQuanXian")%>";
                    var ChongQi = '<%=rb.getString("ChongQi")%>';
                    var Restart = '<%=rb.getString("ChongXinKaiShi")%>'
                    var RiZhi = '<%=rb.getString("RiZhiShouJi")%>';
                    var MiMaChongZhi = '<%=rb.getString("XiuGaiMiMa")%>';
                    var License = '<%=rb.getString("License")%>';
                    var CaoZuo = '<%=rb.getString("CaoZuo")%>';
                    var registerSAS = '<%=rb.getString("SasZhuCe")%>'; 
                    var deregisterSAS = '<%=rb.getString("ZhuXiaoSAS")%>';
                    var GengDuoCaoZuo = '<%=rb.getString("GengDuoCaoZuo")%>';
                    var WeiHuCaoZuo = '<%=rb.getString("Maintenance")%>';
                    var PeiZhiHuiFu = '<%=rb.getString("PeiZhiHuiFu")%>';
                    var RFname = '<%=rb.getString("ShePinCaoZuo")%>';
                    var RFon = '<%=rb.getString("Kai")%>';
                    var RFoff = '<%=rb.getString("Guan")%>';
                    var YouXiaoQi = '<%=rb.getString("YouXiaoQi")%>';
                    var LiuLiangXianZhi = '<%=rb.getString("LiuLiangXianZhi")%>';
                    var FenBuShi = '<%=rb.getString("FenBuShi")%>';
                    var rfText ,rfText1,rfText2,forceRFText,forceRFStatus,
                        rfChild = [];
                    var rizhidisableflag;
                    var rfStatusShowfloag=true;
                    var rfStatus1,rfStatus2;
                    var platformType = rowDatas.platformType,
                        dualCarrierType = rowDatas.dual_carrier_type,
                        moduleType = rowDatas.module_type;
                    
                    var showFlagList={
                            CODE_ENB_REBOOT:false,
                            CODE_ENB_RESET_CONFIG:false,
                            CODE_ENB_LOGS:false,
                            CODE_ENB_SYNCHRONIZE:false,
                            CODE_ENB_HALOB_ENABLE:false,
                            CODE_ENB_RF_ENABLE:false,
                            CODE_ENB_ACTIVE:false,
                            CODE_ENB_SAS_RF_ENABLE:false,
                            CODE_ENB_SAS_ENABLE:false,
                            CODE_ENB_LOCK:false,
                            CODE_ENB_CHANGE_PASSWORD:false,
                            CODE_ENB_INFORMATION:false,
                            CODE_ENB_SETTINGS:false,
                            CODE_ENB_EXPIRY_DATE:false,
                            CODE_ENB_EXPIRY_DATE_LOCK:false,
                            CODE_ENB_DISTRIBUTED:false,
                            CODE_ENB_TRAFFIC_LIMITATION:false,
                        },
                        paramsCode={
                            smallCellCode:idval
                        },
                        maintenanceShow = false,
                        actionShow = false;

                    vm.platformType = platformType;
                    $.ajax({
                        type:'POST',
                        url:'${ctx}/cell/cpeinfos/getENBOperationItem.action',
                        data:paramsCode,
                        async:false,
                        dataType:'json',
                        success:function(data){
                            var operationData = data;
                            if(operationData.Maintenance && operationData.Maintenance.length > 0 ){
                                operationData.Maintenance.map((item)=>{
                                    showFlagList[item] = true;
                                });
                            }
                            if(operationData.Actions && operationData.Actions.length > 0 ){
                                operationData.Actions.map((item)=>{
                                    showFlagList[item] = true;
                                })
                            }
                            if(operationData.others && operationData.others.length > 0 ){
                                operationData.others.map((item)=>{
                                    showFlagList[item] = true;
                                })
                            }
                            
                        },
                    });

                    if(connectStatus != 'Off'){
                        rizhidisableflag = false
                    }else{
                        rizhidisableflag = true;
                    }
                
                    //配置恢复 判断
                    var resetDisableFlag;
                    var tongbuDisableFlag;
                    if(connectStatus != 'Off'){
                        tongbuDisableFlag = false;
                        resetDisableFlag = false;
                    }else{
                        tongbuDisableFlag = true;
                        resetDisableFlag = true;
                    }
                    
                    
                    //判断是否有licence菜单选项
                    var paramLicense={
                            small_cell_code:idval
                        }
                    
                    //判断halob菜单项是否显示  可用
                    //当基站不在线  且不是集中模式   不显示但      非集中模式 且基站在线 菜单可用     非集中模式 且基站不在线 菜单不可用
                    var halobName = "";
                    var disableHalobFlag;
                    var halobSwitch;
                    if(connectStatus != 'Off') disableHalobFlag = false;
                    else disableHalobFlag = true;
                    
                    $.ajax({
                        type:'POST',
                        url:'${ctx}/cell/cpeinfos/checkCellHasLicenseInfo.action',
                        data:paramLicense,
                        async:false,
                        dataType:'json',
                        success:function(data){
                            
                            //控制halob菜单
                            if([0,'0'].includes(data["halobFlag"])){
                                //关闭Halob
                                halobName = '<%=rb.getString("KaiQiHaloB")%>';
                                halobSwitch = 1;
                            }else if(data["halobFlag"] == 1){
                                //开启Halob
                                halobSwitch = 0;
                                halobName = '<%=rb.getString("GuanBiHaloB")%>';
                            }else{
                                //集中模式   集中模式不显示 halob的菜单
                                showFlagList.CODE_ENB_HALOB_ENABLE = false;
                                disableHalobFlag= true;
                                halobName = '<%=rb.getString("HaloBKaiGuan")%>';
                            }
                        },
                    });
                    //判断重启 flag
                    var chongQidisableFlag;
                    if(connectStatus != 'Off'){
                        chongQidisableFlag = false;
                    }else{
                        chongQidisableFlag = true;
                    }
                    

                    //判断激活状态
                    var opStateName = "";
                    var opStateFlag;
                    var opSwitch = ""
                    if(connectStatus != 'Off'){
                        opStateFlag = false;
                    }else{
                        opStateFlag = true;
                    }

                    if(product == 'PM-B4860'){
                         opStateName = '<%=rb.getString("JiHuo")%>/<%=rb.getString("goJiHuo")%>';
                         opSwitch = "0";
                         if(rowDatas.op_state_cell2 && rowDatas.op_state_cell2 == 0){
                             opStateFlag = true;
                         }
                    }else{
                        if(opState == "1" || opState == "1,1"){
                            opStateName = '<%=rb.getString("goJiHuo")%>';
                            opSwitch = "0";
                        }else{
                            opStateName ='<%=rb.getString("JiHuo")%>';
                            opSwitch = "1";
                        }
                    }
                    
                    
                    //基站不在线 RF开关不可操作
                    if(connectStatus != 'Off'){
                        disableRFFlag = false;
                    }else{
                        disableRFFlag = true;
                    }
                    var showCell = false;
                    
                    // 根据RF开关状态，为操作显示的文字赋值 
                    if(rfStatus === '' || rfStatus =='--'){
                        rfStatusShowfloag=false;
                    }else if ( rfStatus == "on" || rfStatus == 1){
                        showCell = false;
                        rfText = RFname + ' '+ RFoff;
                    }else if( rfStatus == "off" || rfStatus === 0){
                        showCell = false;
                        rfText = RFname + ' '+ RFon;
                    }else{
                        showCell = true;
                        rfText = RFname;
                        rfStatus = rfStatus.split(",");

                        rfStatus.map(function(sItem, idx){
                            var rfItemText = 'RF' + (idx+1) + ' ' + (sItem == 'on'? RFoff:RFon);

                            rfChild.push({
                                row: row, 
                                label: rfItemText,
                                id: 432,
                                cls: ' ',
                                rfStatus: sItem,
                                cell_code: idval,
                                show: showCell,
                                cellNumber: idx+1
                            });
                        })
                    }
                    

                    var sasSwitch = ("${sasSwitch}" == "1");
                    var rfForceShowFlag = false;
                    if ( rowDatas.sasEnable == "on"){
                        rfForceShowFlag = true;
                        if (rowDatas.autoOrForceRF == "false" ){
                            forceRFText = "<%= rb.getString("QiangZhiGuanBiRF")%>";
                            forceRFStatus = "true"
                        }else if (rowDatas.autoOrForceRF == "true" ){
                            forceRFText = "<%= rb.getString("SASZiDongKongZhiRF")%>"
                            forceRFStatus = "false"
                        }else if(rowDatas.autoOrForceRF == null || rowDatas.autoOrForceRF == undefined){
                            rfForceShowFlag = false;
                        }
                        
                    }
                    var opStateShowFlag = true,
                        snLastTwoDigits = serialNumber.slice(serialNumber.length-2,serialNumber.length);

                    if(['QA_436Q_DC'].includes(platformType) && snLastTwoDigits == '-2') {
                        opStateShowFlag = false
                    }
                    var maintenanceOpChild = [
                        
                        {row: row, label:PeiZhiHuiFu,id:42,cls:'CODE_ENB_RESET_CONFIG hidden',show:showFlagList.CODE_ENB_RESET_CONFIG,disable: resetDisableFlag,cell_code:idval},
                       
                        
                    ]

                    var scanFlag = ['QA_436Q_CA','QA_436Q_SC','QA_436Q_DC','NEU430_DC','BAIBLQ','MLQ'].includes(platformType) && dualCarrierType != 2;

                    var moreOpChild = [
                           //show为false不显示 disable为false显示禁用
                            {row: row, label: registerSAS, id: 35, cls:'', show: sasSwitch},
                            {row: row, label: deregisterSAS, id: 37, cls:'', show: sasSwitch},
                           
                            {row: row, label:forceRFText,id:48,cls:'CODE_ENB_RF_ENABLE hidden',serialNumber:serialNumber, show:showFlagList.CODE_ENB_SAS_RF_ENABLE && rfForceShowFlag,disable:disableRFFlag,rfStatus: forceRFStatus},
                            //{row: row, label:lockText,id:44,cls:'CODE_ENB_EXPIRY_DATE hidden',cell_code:idval,lock_status:lockStatus,show:!lockOperFlag}
                            
                        ],
                        isSubDevice = false,
                        isUnusable = false;
                    
                    var group1 = [
                        {row: row, label:TongBu,id:31,cls:'CODE_ENB_SYNCHRONIZE hidden',show:showFlagList.CODE_ENB_SYNCHRONIZE,disable: tongbuDisableFlag,cell_code:idval},
                        {row: row, label:'<%=rb.getString("ShouJiBaoWen")%>',id:50,show: collectShow && writableMap['CODE_ENB_TR069_MSG_EXCHANGE'] == true},
                    ]

                    var group2 = [
                        {row: row, label:ChongQi,id:36,cls:'CODE_ENB_REBOOT hidden',show:showFlagList.CODE_ENB_REBOOT,disable: chongQidisableFlag,cell_code:idval,product:product},
                    ]

                    var group3 = [
                        {row: row, label:opStateName,id:41,cls:'CODE_ENB_ACTIVE hidden',show:showFlagList.CODE_ENB_ACTIVE && opStateShowFlag ,disable: opStateFlag,activeStatus:opSwitch,cell_code:idval},
                        {row: row, label:rfText,id:43,cls:'CODE_ENB_RF_ENABLE hidden',cell_code:idval, show:showFlagList.CODE_ENB_RF_ENABLE && rfStatusShowfloag && !showCell,disable:disableRFFlag,rfStatus: rfStatus,cellNumber:''},
                        {row: row, label:rfText,id:431,cls:'CODE_ENB_RF_ENABLE hidden',cell_code:idval, show:showFlagList.CODE_ENB_RF_ENABLE && rfStatusShowfloag && showCell,disable:disableRFFlag,
                            child: rfChild
                        },
                        {row: row, label:halobName,id:40,cls:'',show: vm.isMonitorWritable && showFlagList.CODE_ENB_HALOB_ENABLE, disable: disableHalobFlag,cell_code:idval,halob_switch:halobSwitch},
                        //{row: row, label:Restart,id:52, show: is_super_user == 'true' && row.stun_reboot == 1, cell_code:idval, product:product},
                    ]

                    var group4 = [
                        {row: row, label:RiZhi,id:32,cls:'CODE_ENB_LOGS hidden',show:showFlagList.CODE_ENB_LOGS,disable: rizhidisableflag},
                    ]
          
                    
                    if(rowDatas.have_connected == 2) {// 是否真实可用站
                        //moreOpChild = [];
                        group1 = [] ;
                        group2 = [] ;
                        group3 = [] ;
                        group4 = [] ;
                        isUnusable = true;
                    }
                    var id4param = idval;
                    // 新类型QA_436Q_DC的处理
                    if(['QA_436Q_CA','QA_436Q_SC','QA_436Q_DC','NEU430_DC'].includes(platformType) && dualCarrierType == 2) {// 辅波查看不可操作，更多操作只有射频
                        isSubDevice = true;
                        isUnusable = false;
                        id4param = id4param.substr(0,id4param.length-2);
                    }
                    // 4860辅站
                    if(['Intel_CR_DC','MLN_DC'].includes(platformType) && dualCarrierType == 2) {
                        group3 = [
                            {row: row, label:opStateName,id: 411,cls:'CODE_ENB_ACTIVE hidden',show:showFlagList.CODE_ENB_ACTIVE,disable: opStateFlag,activeStatus:opSwitch,cell_code:idval},
                            {row: row, label:rfText,id:43,cls:'CODE_ENB_RF_ENABLE hidden',cell_code:idval, show:showFlagList.CODE_ENB_RF_ENABLE && rfStatusShowfloag && !showCell,disable:disableRFFlag,rfStatus: rfStatus,cellNumber:''}
                        ];
                        isSubDevice = true;
                        isUnusable = false;
                    }
                    // 4860 CA - 多小区   2+4 产品类型也显示多小区激活状态操作
                    if(['Intel_CR_CA','Intel_CR_TC','MLN_CA', 'BM'].includes(platformType)) {
                        var activeRow = group3.filter(function(m){ return m.id == '41';});
                        
                        if(activeRow && activeRow.length) {
                            var cellChildren = [];
                            
                            // 处理4G小区
                            if(!['0','1','',0,1].includes(opState)) {
                                opState.split(',').forEach(function(s, idx) {
                                    var cellPrefix = platformType == 'BM' ? 'LTE Cell' : 'Cell',
                                        opIdxName = cellPrefix + (idx+1) + '  <%=rb.getString("JiHuo")%>',
                                        opIdxSwitch = '1';
                                    
                                    if(s == "1"){
                                        opIdxName = cellPrefix + (idx+1) + '  <%=rb.getString("goJiHuo")%>';
                                        opIdxSwitch = "0";
                                    }
                                    
                                    cellChildren.push({
                                        row: row, 
                                        label: opIdxName, 
                                        id: (('41' + idx) - 0), 
                                        cls: 'CODE_ENB_ACTIVE hidden',
                                        show: showFlagList.CODE_ENB_ACTIVE,
                                        disable: opStateFlag,
                                        activeStatus: opIdxSwitch,
                                        cell_code: idval, 
                                        cellNumber: (idx+1)
                                    });
                                });
                            }
                            
                            // 处理GSM小区 (仅BM平台)
                            if(platformType == 'BM' && !['0','1','',0,1].includes(gsmOpState)) {
                                gsmOpState.split(',').forEach(function(s, idx) {
                                    var gsmOpIdxName = 'GSM Cell' + (idx+1) + '  <%=rb.getString("JiHuo")%>',
                                        gsmOpIdxSwitch = '1';
                                    
                                    if(s == "1"){
                                        gsmOpIdxName = 'GSM Cell' + (idx+1) + '  <%=rb.getString("goJiHuo")%>';
                                        gsmOpIdxSwitch = "0";
                                    }
                                    
                                    cellChildren.push({
                                        row: row, 
                                        label: gsmOpIdxName, 
                                        id: (('42' + idx) - 0), 
                                        cls: 'CODE_ENB_ACTIVE hidden',
                                        show: showFlagList.CODE_ENB_ACTIVE,
                                        disable: opStateFlag,
                                        activeStatus: gsmOpIdxSwitch,
                                        cell_code: idval, 
                                        cellNumber: (idx+1),
                                        cellType: 'gsm'
                                    });
                                });
                            }
                            
                            // 统一设置子菜单
                            if(cellChildren.length > 0) {
                                activeRow[0].child = cellChildren;
                                activeRow[0].disable = false;
                            }
                        }
                    }

                    var group3Flag = false;
                   for(var i=0;i<group3.length;i++){
                        if(group3[i].show){
                            group3Flag = true;
                            break;
                        }
                    }
                    
                    
                    vm.menus = [
                            {row: row,cls:'',show:group1.length>0 && (collectShow || showFlagList.CODE_ENB_SYNCHRONIZE),
                                child: group1, disable: isUnusable
                            },
                            {row: row, cls:'',show:group2.length>0 && showFlagList.CODE_ENB_REBOOT,
                                child: group2, disable: isUnusable
                            },
                            {row: row,cls:'',show: group3.length>0 && group3Flag,
                                child: group3, disable: isUnusable
                            },
                            {row: row,cls:'',show: group4.length>0 && (showFlagList.CODE_ENB_LOGS || scanFlag),
                                child: group4, disable: isUnusable
                            },
                        ];
                    
                    vm.$nextTick(function(){
                        document.body.click();
                        vm.$refs.menu.show(evt);
                    })
                },
                loadSpecialTab(url, params) {
                    $('#enbSetting_slide_body').load(url,params);
                },
                menuClick(item) {
                    var vm = this,
                        row = item.row;

                    var rowCode = row.small_cell_code;

                    switch(item.id){
                    case 1: //信息
                        goCellDetailParamInfoWin("enbStatistics", row.small_cell_code, row.connection_status, row.CELL_IDENTITY);
                        
                        break;
                    case 2://设置
                    	if(row.platformType.indexOf('436Q')>=0 || row.platformType.indexOf('BAIBLQ')>=0 || row.platformType.indexOf('MLQ')>=0){
                    		goSettingPanel(row.small_cell_code, row.serial_number, row.connection_status, row.dual_carrier_type);
                    	}else{
                    		halobSwitchFlag = row.halob_flag;
                            jumpToSetting(rowCode, item.label, row.platform_flag);
                    	}
                        break;
                    case 3://操作
                        
                        break;
                    case 4://操作
                        
                        break;
                    case 31://同步
                        //refreshCell(item.cell_code);
                        openSyncDialog(item.cell_code);
                       
                        break;
                    case 32://日志
                        confirmImmediateCollectLogFile('queryCollectPopdiv');
                        
                        break;
                    case 35://sas注册
                        registerSas();
                        
                        break;
                    case 36://重启
                        cellReboot(item.cell_code,row.product);
                        
                        break;
                    case 37://deregister
                        deregisterSAS();
                        
                        break;
                    case 40://Halob开启关闭
                        var cellCode = item.cell_code;
                        var halobSwitch = item.halob_switch;
                        openCloseHalob(cellCode,halobSwitch);//开启关闭Halob操作
                        
                        break;
                    case 41://激活状态
                        var cellCode = item.activeStatus;
                        var smallcellCode =  item.cell_code;
                        if(row.product == 'PM-B4860'){
                            vm.activeCellLialogLoading = true;
                            axios.post('${ctx}/pm/nxp/getSlotCellInfos.action',stringify({smallCellCode:smallcellCode})).then(function(response){
                                let data = response.data
                                if(data){
                                    vm.cellInfosList = vm.activeCellsDataFot(data);
                                    vm.activeCellLialogLoading = false;
                                }
                            }).catch(function(error){})
                            vm.showActiveCellDialog = true;
                        }else{
                            activeOpStatus(cellCode,smallcellCode);//开启关闭Halob操作
                        }
                        break;
                    case 410://激活状态 - CA cell1小区
                        var cellCode = item.activeStatus,
                            smallcellCode =  item.cell_code,
                            cellNumber = item.cellNumber;
                        activeOpStatus(cellCode,smallcellCode,cellNumber);//开启关闭Halob操作
                        
                        break;
                    case 411://激活状态 - CA cell2小区
                        var cellCode = item.activeStatus,
                            smallcellCode =  item.cell_code,
                            cellNumber = item.cellNumber;
                        activeOpStatus(cellCode,smallcellCode,cellNumber);//开启关闭Halob操作
                        
                        break;
                    case 412://激活状态 - CA cell3小区
                        var cellCode = item.activeStatus,
                            smallcellCode =  item.cell_code,
                            cellNumber = item.cellNumber;
                        activeOpStatus(cellCode,smallcellCode,cellNumber);//开启关闭Halob操作
                        
                        break;
                    case 420://激活状态 - GSM cell1小区
                        var cellCode = item.activeStatus,
                            smallcellCode =  item.cell_code,
                            cellNumber = item.cellNumber;
                        activeOpStatus(cellCode,smallcellCode,cellNumber,'gsm');//GSM小区激活/去激活操作
                        break;
                    case 421://激活状态 - GSM cell2小区
                        var cellCode = item.activeStatus,
                            smallcellCode =  item.cell_code,
                            cellNumber = item.cellNumber;
                        activeOpStatus(cellCode,smallcellCode,cellNumber,'gsm');//GSM小区激活/去激活操作
                        break;
                    case 422://激活状态 - GSM cell3小区
                        var cellCode = item.activeStatus,
                            smallcellCode =  item.cell_code,
                            cellNumber = item.cellNumber;
                        activeOpStatus(cellCode,smallcellCode,cellNumber, 'gsm');//GSM小区激活/去激活操作
                        break;
                    case 42://恢复默认配置
                        var smallcellCode =  item.cell_code
                        configReset(smallcellCode);
                        
                        break;
                    case 43://RF 
                        var smallcellCode =  item.cell_code
                        setRFStatus(smallcellCode, item.rfStatus);
                        
                        break;
                    case 5:
                        var smallcellCode =  item.cell_code
                        setEffectPeriod();
                        
                        break;
                    //case 44:
                        //var smallcellCode = item.cell_code;
                        //var lockStatus = item.lock_status;
                        //setLockStatus(smallcellCode,lockStatus);
                        
                        //break;
                    case 431:
                        
                        break;
                    case 432://RF 关闭
                        var smallcellCode =  item.cell_code
                        setRFStatus(smallcellCode,item.rfStatus,item.cellNumber)
                        
                        break; 
                    case 48://autoOrForceRF
                        var serialNumber =  row.serial_number;
                        setForceRFStatus(serialNumber, item.rfStatus)
                        
                        break; 
                    case 'limit': // Limitation
                        setLimitation(rowCode);
                        
                        break; 
                    case 6://分布式基站
                        distributeSn("enbDistribute",row.small_cell_code);
                        
                        break;
                    
                    case 50://收集报文
                        vm.showCollectMessage(row);
                        
                        break;
                    case 52://基站Restart
                            vm.restartEnb(row);

                            break;
                    }
                },
                restartEnb(row) {
                    var vm = this,
                        url = '${ctx}/cell/cpeinfos/stunReboot.action',
                        params = {
                            smallCellCodes: row.small_cell_code
                        };

                    axios.post(url, stringify(params)).then(function(res){
                        var data = res.data;

                        if(data.success == true) {
                            vm.$message({
                                message: '<%=rb.getString("XiaFaChengGongZhuangTai")%>',
                                type: 'success',
                            });
                        }else {
                            vm.$message({
                                message: data.msg,
                                type: 'error',
                            });
                        }
                    });
                },
                goPeriod() {
                    var vm = this;

                    vm.periodslide.url = '${ctx}/cell/cpeinfos/toCellValidity.action';

                    vm.$refs.period.showSlide();
                },
                goDistribute(code) {
                    var vm = this;
                    var url = '${ctx}/cell/nxp/toCellInfo.action?smallCellCode='+code;

                    vm.distributeslide.url = url;
                    vm.$refs.distribute.showSlide();
                },
                handerClose() {
                    this.$refs.menu.hide();
                },
                closeInfo() {
                    this.$refs.info.hide();
                },
                goSettingSlide(code, sn, connStatus, carrierType){
                	var vm = this;
                    var url = '${ctx}/cell/quicksettings/goSettingParamPage.action';

                    vm.enbSettingSlide.url = url;
                    vm.$refs.setting.showSlide(function(){
                    	eventBus.$emit('enb-set',code, sn, connStatus, carrierType);
                    });
                },
                cancelSetting(){
                	eventBus.$emit('cancel-set')
                },
                saveSetting(){
                	eventBus.$emit('save-set')
                },
                activeCellsDataFot(data){  // 板卡小区格式化
                    var cellInfosList = [
                        {label:'1',cellItem:[]},
                        {label:'2',cellItem:[]},
                        {label:'3',cellItem:[]},
                        {label:'4',cellItem:[]},
                    ];
                    data.map((item,index)=>{
                        item.oldStatus = item.activeStatus;
                        if(item.cardId == '1'){
                            cellInfosList[0].cellItem.push(item)
                        }else if(item.cardId == '2'){
                            cellInfosList[1].cellItem.push(item)
                        }else if(item.cardId == '3'){
                            cellInfosList[2].cellItem.push(item)
                        }else if(item.cardId == '4'){
                            cellInfosList[3].cellItem.push(item)
                        }
                    });
                    cellInfosList.map((item,index,arr)=>{
                        if(item.cellItem.length>0){
                            item.cellItem = item.cellItem.sort((a,b)=>{
                                return parseInt(a.cellIndex) - parseInt(b.cellIndex)
                            })
                        }
                        
                    })
                    return cellInfosList
                },
                activeCellSubmit(){ // PM-B4860基站 激活/取消激活
                    var vm = this,
                        url = '${ctx}/cell/cpeinfos/cellModifyActiveStatus.action',
                        params = {
                            small_cell_code: vm.selectedRow.small_cell_code, // vm.selectedRow.small_cell_code
                            cellNumber:''
                        },
                        cellNumber = [];

                    vm.cellInfosList.map((item,index)=>{
                        if(item.cellItem.length>0){
                            item.cellItem.map((items,idx)=>{
                                cellNumber.push({cellIndex:items.cellIndex,status:items.activeStatus})
                            })
                        }
                    })
                    cellNumber = JSON.stringify(cellNumber);
                    params.cellNumber = cellNumber;
                    axios.post(url,stringify(params)).then(function(response){
                        let data = response.data
                        if(data["success"]){
                            vm.$message.success('<%=rb.getString("ChengGong")%>')
                            vm.refreshList();
                             vm.showActiveCellDialog = false;
                        }else{
                            vm.$message.error(data.message) //错误提示信息 
                        }
                    }).catch(function(error){})
                },
                closeActiveCellDialog(){ // 关闭 PM-B4860基站 激活/取消激活弹窗
                    var vm = this;
                    vm.cellInfosList = [];
                    vm.showActiveCellDialog = false;
                },
                cellActivePopoverShow(row){ // 激活信息Popover 打开事件
                    var vm = this;

                    vm.cellActivePopoverLoading = true;
                    axios.post('${ctx}/pm/nxp/getSlotCellInfos.action',stringify({smallCellCode:row.small_cell_code})).then(function(response){
                        let data = response.data
                        if(data){
                            vm.activeCellsDataList = vm.activeCellsDataFot(data);
                            vm.cellActivePopoverLoading = false;
                        }
                    }).catch(function(error){})
                },
                cellActivePopoverHide(){ // 激活信息Popover 关闭事件
                    var vm = this;
                    vm.activeCellsDataList = [];
                },
                halodCellPopoverShow(row){ // HaloD cell信息Popover 打开事件
                    var vm = this;
                    //  vm.halodCellDataList = [{serialNumber:'12312312312321',form:'0',lockStatus:'1'},{serialNumber:'120555012546554668',form:'1',lockStatus:'1'}];
                    vm.halodCellPopoverLoading = true;
                    axios.post('${ctx}/cell/halod/queryHalodRelationInfo.action',stringify({serialNumber:row.serial_number})).then(function(response){
                        let data = response.data
                        if(data){
                            vm.halodCellDataList = data.snList;
                            vm.halodCellPopoverLoading = false;
                        }
                    }).catch(function(error){})
                },
                halodCellPopoverHide(){ // HaloD cell信息Popover 关闭事件
                    var vm = this;
                    vm.halodCellDataList = [];
                },
                lockBtnClick(type,row){// halod 锁定与解锁事件
                    var vm = this;
                        urls='',
                        masterSerialNumberList=[],
                        snList = [],
                        params={
                            operatorCode:operatorCodeGloab,
                            masterSerialNumber:''
                        },
                        subCellList = JSON.parse(JSON.stringify(vm.halodCellDataList));
                    subCellList.map((item,index)=>{
                        snList.push(item.serialNumber);
                        if(item.form == '0'){
                            masterSerialNumberList.push(item.serialNumber)
                        }
                    })
                    params.masterSerialNumber = masterSerialNumberList.join(',');
                    if(type == 'lock'){
                        urls = '${ctx}/cell/halod/saveHaloDRelation.action';
                        params.slaveSerialNumber = snList.join(',');
                    }else{
                        urls = '${ctx}/cell/halod/releaseHaloDRelation.action';
                    }
                    axios.post(urls,stringify(params)).then(function(response){
                        var data = response.data;
                        var message = '<%=rb.getString("ChengGong")%>';
                        if(data["success"]){
                            vm.$message({
                                message:message,
                                type:'success',
                            })
                            vm.refreshList();
                        }else{
                            vm.$message.error(data["message"])
                        }
                    })
                },
                filterHalodClick(){// halod 过滤同锁定关系设备
                    var vm = this,
                        snList = [];
                    vm.halodCellDataList.map((item,index)=>{
                        snList.push(item.serialNumber);
                    })
                    queryVue.reseteNBQuery();
                    vm.queryParams.search_text = '';
                    vm.queryParams.halodSerialNumbers = snList.join(',');
                },
                // 打开已选弹窗
                openBulkSelectTable(){
                    var vm = this;
                    vm.bulkSelectShow = true
                },
                // 关闭已选弹窗
                closeBulkSelectTable(){
                    var vm = this;
                    vm.bulkSelectShow = false;
                },
                // 设备已选表格 清空事件
                clearBulkSelected(){
                    var vm = this;
                    vm.selectedRows = [];
                    if (vm.$refs.xTable) {
                        vm.$refs.xTable.clearCheckboxRow();
                    }
                },
                // 设备已选表格 单个删除事件
                delBulkSelected(rows){
                    var vm = this,
                        rowKey = 'serial_number';
                    vm.selectedRows = vm.selectedRows.filter((items)=>{
                        return items[rowKey] != rows[rowKey]
                    });
                    // vxe-table取消选中
                    if (vm.$refs.xTable) {
                        vm.$refs.xTable.setCheckboxRow(rows, false);
                    }
                },
                // 判断小区状态 RF状态 显示  1 全部在线 2 部分在线 3 全部不在线
                judgeActiveStatusFat(value){
                    var status = '1',
                        states = (value+'').split(',');
                    if((states.indexOf('0') >= 0  && states.indexOf('1') >= 0) || (states.indexOf('off') >= 0  && states.indexOf('on') >= 0)){
                        status = '2'
                    }else if((states.indexOf('0') >= 0  && states.indexOf('1') < 0) || (states.indexOf('off') >= 0  && states.indexOf('on') < 0)){
                        status = '3'
                    }else if((states.indexOf('0') < 0  && states.indexOf('1') >= 0) || (states.indexOf('off') < 0  && states.indexOf('on') >= 0)){
                        status = '1'
                    }
                    return status
                },
                // 判断小区装填 RF状态  在线数 与 总数
                judgeActiveNumOrAllNumFat(value,type){
                    var activeNum = [],
                        allNum = [],
                        states = (value+'').split(','),
                        val = 0;
                    states.map((item,index)=>{
                        if(item == '1' || item == 'on'){
                            activeNum.push(item)
                        }
                        allNum.push(item)
                    })

                    if(type == 'active'){
                        val = activeNum.length
                    }else{
                         val = allNum.length
                    }
                    return val
                },
                // 生成 小区状态 RF状态 集合
                parseCellAndRfStatus(value){
                    var states = (value+'').split(',');
                    return states
                },
                parseMMEOnNum(str){
                    var onNum = [],
                        list = [];
                    if(str) {
            		    list = eval('('+str+')');
                    }else {
                        list = [];
                    }
                    list.map((item,index)=>{
                        if(item.status == '1'){
                            onNum.push(item)
                        }
                    })
                    return onNum.length
                },
                init(){
                    var vm = this;
                    if(vm.gsmEnable == 'true'){
                        $("#gsmMonitor").load('${ctx}/cell/cpeinfos/toGSMMonitorPages.action',function(data){
                            $.parser.parse(this);
                        });	
                    }
                    //vm.getCustomLabelData();
                    // vxe-table 初始数据加载
                    vm.loadTableData();
                },
                handleClick(tab) {
                    var vm = this;

                    if(tab.name == 'GSM') {
                        if(vm.gsmEnable == 'true') {
                            gsmvm.initTb();
                        }
                    }else {
                        vm.loadTableData();
                    }
                }

            },
            created() {
                var vm = this;

                axios.post('${ctx}/system/column/setting/load/1').then(function(res){
                    var data = res.data.data||{},
                        sortCodes = (data.sortColumn||'').split(','),
                        sortList = vm.columns;

                    vm.showProps = (data.showColumn||'').split(',');
                    vm.sortColumns = sortCodes;

                    vm.columns = sortList.sort(function(n, m) {
                        var idxn = sortCodes.indexOf(n.field)==-1?100:sortCodes.indexOf(n.field),
                            idxm = sortCodes.indexOf(m.field)==-1?100:sortCodes.indexOf(m.field);

                        return idxn - idxm;
                    });

                    // vxe-table 列排序（如果需要动态排序，可以通过 loadColumn 方法）
                    // 注释掉旧的el-ctable列排序逻辑
                    /*
                    var tbStates = vm.$refs.list.$refs.ctableInner.store.states,
                        storeCols = tbStates.columns;

                    tbStates._columns = storeCols.sort(function(n, m) {
                        var idxn = sortCodes.indexOf(n.property)==-1?100:sortCodes.indexOf(n.property),
                            idxm = sortCodes.indexOf(m.property)==-1?100:sortCodes.indexOf(m.property);

                        if( [undefined,'enb_operation','connection_status','alarm','serial_number','host_name'].includes(n.property) ) {
                            idxn = 0;
                        }
                        if( [undefined,'enb_operation','connection_status','alarm','serial_number','host_name'].includes(m.property) ) {
                            idxm = 0;
                        }

                        return idxn - idxm;
                    });

                    vm.$refs.list.$refs.ctableInner.store.updateColumns();

                    */
                    loadHTML(document.querySelector('#toolbar_tableHomeCellList'),{
                        url: '${ctx}/cell/cpeinfos/toEnbQuery.action',
                        success: function() {
                            loadHTML(document.querySelector('#showOrHideItem'),{
                                url: '${ctx}/cell/cpeinfos/toCellSort.action'
                            });
                            closeLoading();
                        }
                    });

                    //setTimeout(function(){
                        vm.tbURL = '${ctx}/cell/cpeinfos/queryCpeInfosList.action?monitor=1';
                    //},100);
                });
            },
            mounted() {
                // 排序
                var vm = this,
                    sortCodes = enbShowCols.split(','),
                    sortList = vm.columns;
                vm.init();
                
                eventBus.$on('cancel-enb-setting',vm.closeSettingPage);
                // vm.columns = sortList.sort(function(n, m){
                //     var idxn = sortCodes.indexOf(n.field)==-1?100:sortCodes.indexOf(n.field),
                //         idxm = sortCodes.indexOf(m.field)==-1?100:sortCodes.indexOf(m.field);

                //     return idxn - idxm;
                // });

                //加载 新建设备组及 导入功能
                // loadHTML(document.querySelector('#addDeviceWarp'),{
                //     url: '${ctx}/cell/cpeinfos/addOrImportDevice.action',
                //     success: function() {
                //     }
                // });
            }
        });
    </script>
    <script>
        var enbtopoLoaded = false,
            /* 筛选展示逻辑处理 */
            filterActiveAlarmUrl = '${ctx}/fault/view/queryViewPageList.action',	
            filterHisAlarmUrl = '${ctx}/fault/view/queryViewPageList.action',
            isInfoAlarmVisible = false;
        var rules = [
                {
                    title: '<%=rb.getString("YanZhongChengDu")%>',
                    match: function(code){
                        return code == 'alarm_serverity_value';
                    },
                    action: function(code,e){
                        var key = 'alarmServerity';
                        var isInfoAlarmVisible = $('#alarmTabsHisActive').hasClass('active'); 
                        var params = {};
                        if(isInfoAlarmVisible) params = $('#enbHistoryAlarm').datagrid('options').queryParams;
                        else params = $('#enbActiveAlarm').datagrid('options').queryParams ; 
                        params['source'] = 'enb';
                        /* 配置筛选菜单可选项，可以通过接口获取数据 */
                        var data = [];
                        var url = filterActiveAlarmUrl;
                        if(isInfoAlarmVisible) url = filterHisAlarmUrl;
                        datas = [{text:'Critical',value:31001},{text:'Major',value:31002},{text:'Minor',value:31003},{text:'Warning',value:31004}];
                        datas.map(function(item){
                                    var row = {name:'alarm_serverity',label:item.text,value:item.value};
                                    data.push(row);
                                });
                        
                            data.map(function(item){

                                if(params[key]){
                                    var vals = params[key].split(',');
                                    if(vals.includes(item.value+'')) {
                                        item.checked = true;
                                    }
                                }else{ 
                                    item.checked = true;
                                } 
                            }); 
                            
                            /* 生成筛选菜单 */
                            filterMenu({
                                data: data,
                                fn: function(tips){
                                    tips.css({left:e.x-$('#menuAnimate').width(),top: e.clientY-30});
                                    $('#omc_app_ctn').append(tips);
                                },
                                click: function(values){
                                    params[key] = values;
                                    if(isInfoAlarmVisible) $('#enbHistoryAlarm').datagrid('reload');
                                    else $('#enbActiveAlarm').datagrid('reload');
                                }
                            });
                    }
                }
            ];
		var Kai = '<%=rb.getString("Kai")%>',
			Guan = '<%=rb.getString("Guan")%>',
			yilianjie = '<%=rb.getString("LianJieZhengChang")%>',
			weilainjie = '<%=rb.getString("LianJieDuanKai")%>';
			
        /* 菜单隐藏处理 */
        $('#omc_app_ctn').mousedown(function(){
            try{
                var target = event.target, list = Array.from(target.classList),
                    plist = Array.from(target.parentNode.classList);
                if(!(list.includes('filter-menu') || list.includes('filter-item') || plist.includes('filter-item'))){
                    $('.filter-menu').hide();
                }
            }catch(e){}
        });

        refresh_cellStatusStatistics();

        function loadEnbTopo() {
            if(enbtopoLoaded) return;
            enbtopoLoaded = true;

            // $('#enb_topo_ctner').load('${ctx}/cell/topo/toMonitorTopo.action',function(html){
            //     $.parser.parse(this);
            // });
        }

        function RFStatusFormatter(value, rowData, rowIndex){
            var row = rowData || {};

            if (value == null || value == "") {
                return null;
            }
            if (value == '--') {
                return value;
            }
            
            if (value == "on" || value == 1) {
                    //CA 模式有两个小区； 射频状态要显示两个图标； 辅小区不能编辑，下发射频开启或关闭，只能从主小区下发； 射频状态两个小区相同； 辅小区跟随主小区
                    if(row.platformType == 'Intel_CR_CA' || row.platformType == 'MLN_CA') {// 4860
                    	value = "<div class='rfConnItem' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' type='RF'>"+ Kai +"</div>"
                            + "<div class='mmeDetails MMEDetail'>RF Status : <span class='mmeStauts'></span></div>"
                            + "<div class='rfConnItem' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' type='RF' style='margin-left: 10px;'>"+Kai+"</div>"
                            + "<div class='mmeDetails MMEDetail'>RF Status : <span class='mmeStauts'></span></div>";
                	}else {
                		value = "<div class='rfConnItem' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' type='RF'>"+Kai+"</div>"
                            + "<div class='mmeDetails MMEDetail'>RF Status : <span class='mmeStauts'></span></div>";
                	}
            } else if (value == "off" || value == 0) {
                  //CA 模式有两个小区； 射频状态要显示两个图标； 辅小区不能编辑，下发射频开启或关闭，只能从主小区下发； 射频状态两个小区相同； 辅小区跟随主小区
                    if(row.platformType == 'Intel_CR_CA' || row.platformType == 'MLN_CA') {// 4860
                    	value = "<div class='rfDisconnItem' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' type='RF'>"+Guan+"</div>"
                            + "<div class='mmeDetails MMEDetail'>RF Status : <span class='mmeStauts'></span></div>"
                            + "<div class='rfDisconnItem' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' type='RF' style='margin-left: 10px;'>"+Guan+"</div>"
                            + "<div class='mmeDetails MMEDetail'>RF Status : <span class='mmeStauts'></span></div>";
                	}else {
                		value = "<div class='rfDisconnItem' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' type='RF'>"+Guan+"</div>"
                            + "<div class='mmeDetails MMEDetail'>RF Status : <span class='mmeStauts'></span></div>";
                	}
                    
            } else{
                var rfStatus = value.split(",");
                var rfStatusText ='';
                if(rfStatus[0] == "on" || rfStatus[0] == "1"){
                    rfStatusText = rfStatusText + "<div class='rfConnItem' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' type='RF1'>"+Kai+"</div>";
                }
                if(rfStatus[0] == "off" || rfStatus[0] == "0"){
                    rfStatusText = rfStatusText + "<div class='rfDisConnItem' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' type='RF1'>"+Guan+"</div>";
                }
                if(rfStatus[1] == "on" || rfStatus[1] == "1"){
                    rfStatusText = rfStatusText + "<div class='rfConnItem' style='margin-left:10px;' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' type='RF2'>"+Kai+"</div>";
                }
                if(rfStatus[1] == "off" || rfStatus[1] == "0"){
                    rfStatusText = rfStatusText + "<div class='rfDisConnItem' style='margin-left:10px;' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' type='RF2'>"+Guan+"</div>";
                }
                    
                if(rfStatus.length == 3) {
                	if(rfStatus[2] == "on" || rfStatus[2] == "1"){
                        rfStatusText = rfStatusText + "<div class='rfConnItem' style='margin-left:10px;' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' type='RF3'>"+Kai+"</div>";
                    }
                    if(rfStatus[2] == "off" || rfStatus[2] == "0"){
                        rfStatusText = rfStatusText + "<div class='rfDisConnItem' style='margin-left:10px;' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' type='RF3'> "+Guan+"</div>";
                    }
                    
                    value = rfStatusText + 
                    		"<div class='mmeDetails MME1Detail'>RF1 Status : <span class='mme1Stauts'></span></div>"+ 
                    		"<div class='mmeDetails MME2Detail'>RF2 Status : <span class='mme2Stauts'></span></div>"+ 
                    		"<div class='mmeDetails MMEDetail'>RF3 Status : <span class='mme3Stauts'></span></div>"; 
                }else {
                    value = rfStatusText + "<div class='mmeDetails MME1Detail'>RF1 Status : <span class='mme1Stauts'></span></div>"+ "<div class='mmeDetails MME2Detail'>RF2 Status : <span class='mme2Stauts'></span></div>"; 
                } 
            }
            return value;
        }

        function toMMEDetail(ele,index){
            var thisTop = $(ele).offset().top;
            var allHeight = $(document).height();
            var thisLeft = $(ele).offset().left;
            var allWidth = $(document).width();
            
            if((allHeight - thisTop) < 200){
                $(ele).siblings(".mmeDetails").css("top","-55px");
            }
            if((allWidth - thisLeft) < 200){
                $(ele).siblings(".mmeDetails").css("left","-90px");
            }
            var eleClass = $(ele).attr('type'); 
            if(eleClass.includes("RF")){
                var YiLianJie = '<%= rb.getString("KaiQi")%>';
                var WeiLianJie = '<%= rb.getString("GuanBi")%>';
            }
            if(eleClass.includes("MME")){
                var YiLianJie = '<%= rb.getString("MMEYiLianJie")%>';
                var WeiLianJie = '<%= rb.getString("MMEWeiLianJie")%>';
            }
            $(".mmeDetails").hide();
            if(eleClass == 'RF1' || eleClass == 'MME1'){
                $(ele).siblings(".MME1Detail").fadeToggle();
                if(index == 1){
                    $(".mme1Stauts").text(YiLianJie);
                }else if(index == 0){
                    $(".mme1Stauts").text(WeiLianJie);
                }
            }else if(eleClass == 'RF2' || eleClass == 'MME2'){
                $(ele).siblings(".MME2Detail").fadeToggle();
                if(index == 1){
                    $(".mme2Stauts").text(YiLianJie);
                }else if(index == 0){
                    $(".mme2Stauts").text(WeiLianJie);
                }
            }else if(eleClass == 'RF3' || eleClass == 'MME3'){
                $(ele).siblings(".MMEDetail").fadeToggle();
                if(index == 1){
                    $(".mme3Stauts").text(YiLianJie);
                }else if(index == 0){
                    $(".mme3Stauts").text(WeiLianJie);
                }
            }else if(eleClass == 'RF' || eleClass == 'MME'){
                $(ele).siblings(".MMEDetail").fadeToggle();
                if(index == 1){
                    $(".mmeStauts").text(YiLianJie);
                }else if(index == 0){
                    $(".mmeStauts").text(WeiLianJie);
                }
            }	
        }

        function hideMMEDetail() {
            $(".mmeDetails").fadeOut(100);
        }

        function ueCountsFormatter(value, rowData, rowIndex) {
            var hostName = rowData.host_name||'';

            if(value == 0 ){
                return "<a style='color:#000000;text-decoration:none;cursor:default;' href='#'>0</a>"; 
            }else if(value == -1 || value == null){
                return "<a style='color:#000000;text-decoration:none;cursor:default;' href='#'>--</a>"; 
            }else{
                if(['QA_436Q_CA','QA_436Q_SC','QA_436Q_DC', 'Intel_CR', 'BAIBLQ', 'MLN', 'MLQ' ,'MLN_CA','MLN_SC','MLN_DC'].includes(rowData.platformType)) {
                    return "<a style='color:#1DA3FC;text-decoration:underline' href='#' onclick='getueCountsData(&quot;" + rowData.small_cell_code + "&quot;,&quot;" + hostName + "&quot;,&quot;" + rowData.serial_number + "&quot;,\"enodeb_ueCounts\",true)'>"+value+"</a>"; 
                }else if(rowData.ca_flag == "1" || rowData.platform_flag == "1"){//CA的站不支持UE数详细信息展示 ca_flag==1 表示为CA站
                    return value;
                }else{
                    return "<a style='color:#1DA3FC;text-decoration:underline' href='#' onclick='getueCountsData(&quot;" + rowData.small_cell_code + "&quot;,&quot;" + hostName + "&quot;,&quot;" + rowData.serial_number + "&quot;,\"enodeb_ueCounts\")'>"+value+"</a>"; 
                }
            }
            
        }
		
        function euruCountsFormatter(value, rowData, rowIndex) {
            var str,connCount,totalCount,arr;
            
            if(value == '--' || value == '' || value == null){
            	str = '--'; 
            }else if(value == "0"){
            	str = '0'; 
            }else{
            	arr = value.split("/");
            	connCount = parseInt(arr[0]);
            	totalCount = parseInt(arr[1]);
            	
            	if(connCount == 0 || totalCount == 0){
            		str="<div class='redColor'>"+ connCount +"/" + totalCount +"</div>"
            	}else if(connCount < totalCount){
            		str="<div class='redColor'>"+ connCount +"/" + totalCount +"</div>"
            	}else{
            		str="<div class=''>"+ connCount +"/" + totalCount +"</div>"
            	}
            	
            }
            return str;
        }
        
        
        function getueCountsData(code,cellName,sn,divId,is436q) {
            var enb_code = code,snNumber= sn,cellName = cellName,
                params = {
                    enb_code: enb_code
                },
                ueCounts_title = '<%=rb.getString("UEShu")%>(<%=rb.getString("XiaoZhanBianMa")%>:' + snNumber  + ','+ '<%=rb.getString("HostName")%>:'+cellName + ')';

            enbvm.ueslide.title = ueCounts_title;
            enbvm.ueslide.height = '100%';
            enbvm.ueslide.is436q = is436q;

            // 获取ue数据
            $.post("${ctx}/system/device/enb/uedata/getENBUeStatisticsDataList.action",params,function(data){
                if(Array.isArray(data)){
                    enbvm.$refs.ueCount.showSlide();
                    enbvm.ueslide.data = data? data:[];
                    
                    if('ue_s1ap_id' in data[0]){
                   	 	enbvm.ueS1apId = true;
                    }else{
                   	 	enbvm.ueS1apId = false;
                    }
                    if('mme_s1ap_id' in data[0]){
                   		enbvm.mmeS1apId = true;
                    }else{
                   	 	enbvm.mmeS1apId = false;
                    }
                }else{
                    toast("<%=rb.getString("BuZhiChiUEShuZuanQu")%>","#omc_app_ctn","",true);
                }
            },"json");
        }

        function cpeCountsFormatter(value, rowData, rowIndex){
            if(value == -1 || value == null){
                return "<a style='color:#000000;text-decoration:none;cursor:default;' href='#'>--</a>"; 
            }else if(value === 0 || value === '0') {
                return value;
            }else{
                var is436Q = ['QA_436Q_CA','QA_436Q_SC','QA_436Q_DC'].includes(rowData.platformType);
                
                return "<a style='color:#1DA3FC;text-decoration:underline' href='#' onclick='getueCpeCountsData(&quot;" 
                        + rowData.small_cell_code + "&quot;,&quot;" + rowData.CELL_IDENTITY + "&quot;,&quot;" + rowData.PHYCELLID 
                        + "&quot;,&quot;" + rowData.EARFCNDLINUSE + "&quot;,&quot;" + rowData.serial_number + "&quot,&quot;" 
                        + rowData.host_name + "&quot,\"enodeb_cpe_connect\","+is436Q+")'>"+value+"</a>"; 
            }
        }

        function getueCpeCountsData(code,eci,pci,earfcn,sn,cellname,divId,is436Q) {
            var url = '${ctx}/cell/ap/toUEDetailPage.action',
                obj ={
                    smallCellCode: code,
                    eci: eci,
                    pci: pci,
                    earfcn: earfcn
                };

            enbvm.cpeslide.url = url;
            
            window.sessionStorage.setItem('ueSmallCellCode',code);
            window.sessionStorage.setItem('ueEci',eci);
            window.sessionStorage.setItem('uePci',pci);
            window.sessionStorage.setItem('ueEarfcn',earfcn);
            window.sessionStorage.setItem('snNumber',sn);
            window.sessionStorage.setItem('cellName',cellname);
            window.sessionStorage.setItem('is436Q',is436Q);

            enbvm.$refs.cpeCount.showSlide(obj, function() {
                $("#tabAlarmCli").click();
            });
        }

        function earfcnFormatter(value, rowData, rowIndex){
            if(value == null){
                return "";
            }
            var EARFCN = value;//正常显示的频点值  还需要将此值转换成频率
            var frequency = 0;
            if (EARFCN >= 36000 && EARFCN <= 36199) { //tdd-band 33
                frequency = 1900 + 0.1 * (EARFCN - 36000);
            } else if (EARFCN >= 36200 && EARFCN <= 36349) { //tdd-band 34
                frequency = 2010 + 0.1 * (EARFCN - 36200);
            } else if (EARFCN >= 36350 && EARFCN <= 36949) { //tdd-band 35
                frequency = 1850 + 0.1 * (EARFCN - 36350);
            } else if (EARFCN >= 36950 && EARFCN <= 37549) { //tdd-band 36
                frequency = 1930 + 0.1 * (EARFCN - 36950);
            } else if (EARFCN >= 37550 && EARFCN <= 37749) { //tdd-band 37
                frequency = 1910 + 0.1 * (EARFCN - 37550);
            } else if (EARFCN >= 37750 && EARFCN <= 38249) { //tdd-band 38
                frequency = 2570 + 0.1 * (EARFCN - 37750);
            } else if (EARFCN >= 38250 && EARFCN <= 38649) { //tdd-band 39
                frequency = 1880 + 0.1 * (EARFCN - 38250);
            } else if (EARFCN >= 38650 && EARFCN <= 39649) { //tdd-band 40
                frequency = 2300 + 0.1 * (EARFCN - 38650);
            } else if (EARFCN >= 39650 && EARFCN <= 41589) { //tdd-band 41
                frequency = 2496 + 0.1 * (EARFCN - 39650);
            } else if (EARFCN >= 41590 && EARFCN <= 43589) { //tdd-band 42
                frequency = 3400 + 0.1 * (EARFCN - 41590);
            } else if (EARFCN >= 43590 && EARFCN <= 45589) { //tdd-band 43
                frequency = 3600 + 0.1 * (EARFCN - 43590);
            } else if (EARFCN >= 18000 && EARFCN <= 18599) { //fdd-band 1
                frequency = 1920 + 0.1 * (EARFCN - 18000);
            } else if (EARFCN >= 0 && EARFCN <= 599) {
                frequency = 2110 + 0.1 * (EARFCN - 0);
            } else if (EARFCN >= 18600 && EARFCN <= 19199) { //fdd-band 2
                frequency = 1850 + 0.1 * (EARFCN - 18600);
            } else if (EARFCN >= 600 && EARFCN <= 1199) {
                frequency = 1930 + 0.1 * (EARFCN - 600);
            } else if (EARFCN >= 19200 && EARFCN <= 19949) { //fdd-band 3
                frequency = 1710 + 0.1 * (EARFCN - 19200);
            } else if (EARFCN >= 1200 && EARFCN <= 1949) {
                frequency = 1805 + 0.1 * (EARFCN - 1200);
            } else if (EARFCN >= 19950 && EARFCN <= 20399) { //fdd-band 4
                frequency = 1710 + 0.1 * (EARFCN - 19950);
            } else if (EARFCN >= 1950 && EARFCN <= 2399) {
                frequency = 2110 + 0.1 * (EARFCN - 1950);
            } else if (EARFCN >= 20400 && EARFCN <= 20649) { //fdd-band 5
                frequency = 824 + 0.1 * (EARFCN - 20400);
            } else if (EARFCN >= 2400 && EARFCN <= 2649) {
                frequency = 869 + 0.1 * (EARFCN - 2400);
            } else if (EARFCN >= 20650 && EARFCN <= 20749) { //fdd-band 6
                frequency = 830 + 0.1 * (EARFCN - 20650);
            } else if (EARFCN >= 2650 && EARFCN <= 2749) {
                frequency = 875 + 0.1 * (EARFCN - 2650);
            } else if (EARFCN >= 20750 && EARFCN <= 21449) { //fdd-band 7
                frequency = 2500 + 0.1 * (EARFCN - 20750);
            } else if (EARFCN >= 2750 && EARFCN <= 3449) { 
                frequency = 2620 + 0.1 * (EARFCN - 2750);
            } else if (EARFCN >= 3450 && EARFCN <= 3799) { //fdd-band 8
                frequency = 925 + 0.1 * (EARFCN - 3450);
            } else if (EARFCN >= 5010 && EARFCN <= 5179) { //fdd-band 12
                frequency = 729 + 0.1 * (EARFCN - 5010);
            } else if (EARFCN >= 5180 && EARFCN <= 5279) { //fdd-band 13
                frequency = 746 + 0.1 * (EARFCN - 5180);
            } else if (EARFCN >= 5730 && EARFCN <= 5849) { //fdd-band 17
                frequency = 734 + 0.1 * (EARFCN - 5730);
            } else if (EARFCN >= 6150 && EARFCN <= 6449) { //fdd-band 20     791-821 6150-6449  
                frequency = 791 + 0.1 * (EARFCN - 6150);
            }else if (EARFCN >= 9210 && EARFCN <= 9659) { //fdd-band 28       758-803 9210-9659
                frequency = 758 + 0.1 * (EARFCN - 9210);
            }else if(EARFCN >= 55240 && EARFCN <= 56740){
                frequency = 3550 + 0.1*(EARFCN - 55240);
            }else if(EARFCN >= 46790 && EARFCN <= 54539){
                frequency = 5150 + 0.1*(EARFCN - 46790);
            }else if(EARFCN >= 63000 && EARFCN <= 63999){
                frequency = 5150 + 0.1*(EARFCN - 63000);
            }else if(EARFCN >= 64000 && EARFCN <= 64999){
                frequency = 5725 + 0.1*(EARFCN - 64000);
            }else if(EARFCN >= 46790 && EARFCN <= 54539){
                frequency = 5150 + 0.1*(EARFCN - 46790);
            }else if(EARFCN >= 63000 && EARFCN <= 63999){
                frequency = 5150 + 0.1*(EARFCN - 63000);
            }else if(EARFCN >= 64000 && EARFCN <= 64999){
                frequency = 5725 + 0.1*(EARFCN - 64000);
            } else {
            //throw new Exception("Please Input the right EARFCN!");
            frequency = "--";
            }
            var showStr = EARFCN.toString()+"("+frequency.toString()+"MHz"+")";
            
            return showStr;
        }

        function getEnbUnuseTimeData(rowData,divId) {
            arrayDate = [];
            arrayDay = [];
            arrayTime = [];

            enbvm.$refs.activeRatio.showSlide(function(){
                try{
                    echarts.getInstanceByDom(document.querySelector('#echart_activeRatio')).resize();
                }catch(e){}
            })
            
            var enb_code = rowData.small_cell_code;
            // 初始化x、y坐标数据
            yArr = [];
            xArr = [];
            
            var param = {
                timeZone:timeZone,
                enb_code:enb_code
            }
            $.post( "${ctx}/system/device/enb/unusetime/getUnuseTimeData.action", param, function(data){
                var availableDayDurationList = data ? (data.availableDayDurationList||[]) : [];
                var unavailableTimeRangeList = (data&&data.unavailableTimeRangList) ? data.unavailableTimeRangList : [];
                var availableDayDurationSort = availableDayDurationList.sort(compare("date"));
                var timeRangeSort = unavailableTimeRangeList.sort(compare("start_time"));
                $.each(availableDayDurationSort, function(index, item) {
                    xArr.push(item.date);
                    yArr.push(item.ailable_duration);
                });
                
                // 处理图表横轴数据，若数据跨月，将月的第一天前显示出月份，并粗体显示
                xArr = xArr.map(function(item) {
                    if (item.substring(8,10) == "01") {
                        return {
                            value: item.substring(5, 10).replace('-','.'),
                            textStyle: {
                                fontWeight: 'bold'
                            }
                        };
                    } else {
                        return {
                            value: item.substring(8, 10),
                            textStyle: {
                                fontWeight:'normal'
                            }
                        };
                    }
                });
                
                // 拼装不可用时间段表格的数据 
                $.each(timeRangeSort, function(index, item) {
                    var startTime = item.start_time;
                    var endTime = item.end_time;
                    var riQiStart = startTime.split(" ")[0];
                    var shiJianStart = startTime.split(" ")[1];
                    var riQiEnd = endTime.split(" ")[0];
                    var shiJianEnd = endTime.split(" ")[1];
                    if (riQiStart != riQiEnd ) {
                        shiJianEnd = "23:59:59";
                    }
                    var timeRangeStr = shiJianStart + "~" + shiJianEnd;
                    var obj = {};
                    obj.days = riQiStart;
                    obj.nousedTime = timeRangeStr;
                    arrayDate.push(obj);
                });
                
                // 创建图表 
                createEchartAvail(availableDayDurationSort);

                enbvm.activeslide.data = arrayDate;
            }, "json");
        }

        function compare(propertyName){
            return function(object1,object2){
                var value1=object1[propertyName];
                var value2=object2[propertyName];
                if(value2<value1) return 1;
                else if(value2>value1) return -1;
                else return 0;
            }
        }

        function createEchartAvail(availableDayDurationSort){
            var enodebChart = echarts.init(document.getElementById("echart_activeRatio"));
            var option = {
                tooltip: {
                    trigger: 'axis',
                    backgroundColor: 'rgba(205,224,232,0.9)',
                    textStyle: {
                        color: "#21608a"
                    },
                    formatter: function(params){
                        var str = "";
                        str += "<div><%=rb.getString("ZiDongBeiFenShiJian")%>:"+availableDayDurationSort[params[0].dataIndex].date+"<br/><%=rb.getString("KeYongFenZhongShu")%>:"+params[0].data+"</div>";
                        return str;
                    }
                },
                xAxis: [{
                    name: "<%=rb.getString("RiQi")%>",
                    type: "category",
                    data: xArr,
                    axisLabel: {
                        show: true,
                        interval: 0
                    },
                    boundaryGap: false
                }],
                yAxis: [{
                    name: "<%=rb.getString("FenZhong")%>",
                    type: "value",
                        axisTick: {
                            show: false
                        }
                }],
                series: [{
                    type: 'line',
                    itemStyle: {
                        normal: {
                            color: '#85B1DE'
                        }
                    },
                    data: yArr,
                    lineStyle: {
                        normal: {
                            color: '#85B1DE'
                        }
                    },
                    showAllSymbol:true
                }],
                grid: { x1: 0, x2: 150, y2: 40, y1: 40 }
            };
            enodebChart.setOption(option);
            // 窗体变化时自适应
            $(window).on('resize',function(){
                try {
                    setTimeout(function(){enodebChart.resize();},200);
                } catch (e) {}
            })
        }

        function gpsFormatter(value, row, rowIndex,field) {
            var isChanged = row.gps_modify_flag == 1,
                str = '',
                lat = row.gps_latitude||'',
                lng = row.gps_longitude||'',
                height = row.gps_height||'';
            
            if(isChanged) {
                value = row['modify_'+field]==undefined? row['gps_'+field]:row['modify_'+field];
            }
            if(value != undefined && isChanged) {
                str = '<div class="cellNameClass"  onclick="showGPSTip(this)" ><span  title="<%=rb.getString("GPSTongBuTipOne")%>&#10;<%=rb.getString("JingDu")%>: '+ lng +'&nbsp;&nbsp;<%=rb.getString("WeiDu")%>: '+lat+'&nbsp;&nbsp;<%=rb.getString("GaoDu")%>: '+height+'"  class="el-icon el-icon-circle-warning"></span></div><span style="margin-left:5px;">'+value+'</span>'
                        +'<div class="syncNameInfo"> <span class="panel_close" onclick="closeSyncName(this)"></span>'
                        +'<div><%=rb.getString("JingDu")%>: '+lng+'&nbsp;&nbsp;<%=rb.getString("WeiDu")%>: '+lat+'&nbsp;&nbsp;<%=rb.getString("GaoDu")%>: '+height+'</div>'
                        +'<div><%=rb.getString("TongBuGPSTiShi")%></span><div>'
                        +'<div><span class="button_simple" onclick="synchronizeGPS(&quot;'+row.small_cell_code+'&quot;)"><%=rb.getString("QueDing")%></span>' 
                        +'<span class="button_simple white" onclick="closeSyncName(this)"><%=rb.getString("QuXiao")%></span>'
                        +'</div></div>';
            }else {
                str = value;
            }
            return str;
        }

        function jumpToSetting(code,title,platform) {
            title += ' <span style="font-size: 12px;"></span>';
            enbPlatform = platform;

            $('#enbSetting_slide .slidebarTitleContainer .default').html(title).data('old',title);
            $('#enbSetting_slide_body').css({overflow: 'auto'}).html('');
            $('#setting_form_cnt').addClass('loading');
            $('.form_bt_refresh').hide();

            $('#enbSetting_slide').slideDown(500,function(){
                var postData = {smallCellCode: code};
                $('#enbSetting_slide').data('params',postData);
                $('#quick_setting_nav li:first').click();
            });
        }
        
        function settingTabClick(id,code,title,subTitle,platform){
            title = title + ' <span style="font-size: 12px;font-weight: normal;">'+subTitle+'</span>';
            enbPlatform = platform;
            $('#enbSetting_slide .slidebarTitleContainer .default').html(title).data('old',title);
            $('#enbSetting_slide_body').css({overflow: 'auto'}).html('');
            $('#setting_form_cnt').addClass('loading');
            $('.form_bt_refresh').hide();

            var postData = {id: id, smallCellCode: code};
            $('#enbSetting_slide').data('params',postData);
            $.ajax({
                url: '${ctx}/cell/quicksettings/getParamNodeTreeAndData.action',
                data: postData,
                type: 'post',
                dataType: 'json',
                success: function(data){
                    renderData2Dom(data);
                    setTimeout(function(){
                        $('.form_bt_refresh').show();
                    },500);
                },
                error: function(data){
                    $('#setting_form_cnt').removeClass('loading');
                }
            });
        }
        /**
        * 把参数组数据渲染成dom节点
        * @param data{array}: 参数组集合
        **/
        function renderData2Dom(data) {
            $('#enbSetting_slide_body').html('');
            Render.tableCollector = {};
            Render.gRender(data,document.querySelector('#enbSetting_slide_body'));
            setTimeout(function(){
                $('#enbSetting_slide_body .group-title:not(:first) .title-text').click();
                var group = $("#enbSetting_slide_body .form-group");
                group.each(function(){
                	var item = $(this).find(".form-wrap .form-item");
                	var i =1;
                	item.each(function(){
                		if($(this).hasClass("form-double")){
                			$(this).addClass("double"+i);
                			i++;
                		}
                	})
                })
                
                
                
                setTimeout(function(){
                    $('#setting_form_cnt').removeClass('loading');
                },500);
            },0);
        }
        /**
        * 刷新分组数据并渲染
        * @param param{array}: 分组数据的查询参数
        **/
        function refreshRenderData2Dom(param) {
            var slider = $('#enbSetting_slide'),
                postData = slider.data('params');
            
            if(slider.length == 0) return;

            // 重新请求分组数据渲染
            if(param.groupId == postData.id && param.smallCellCode == postData.smallCellCode) {
                enbSettingNewPanelVue.changeMenu(param.smallCellCode,param.groupId,false)
            }
        }
        /**
        * 关闭设置浮层面板
        * @param bool{boolean}: 是否检测数据变动
        **/
        function closeSettingPanel(bool){
            var addEdit = false
            var params = Render.getFormDatas($('#enbSetting_slide_body')),toClose=false;
            if(bool==true){
                if(addEdit){ // 保存后变为true 在点击关闭或者取消就会直接关闭
                    $('#enbSetting_slide').slideUp(500);
                }else{ // 没有点击保存 为false 
                    if(!isEmptyJson(params)) {
                    $.messager.confirm(TISHI,'<%=rb.getString("QueDingLiKaiDangQianYeMian")%>',function(r){
                        if(r) $('#enbSetting_slide').slideUp();
                    }).addClass("seriousConfirm");
                    }else{
                        $('#enbSetting_slide').slideUp(500);
                    }
                }
            }else{
                $('#enbSetting_slide').slideUp(500);
            }
        }
        
        function goCellDetailParamInfoWin(divId,small_cell_code,status,CELL_IDENTITY) {
            var url = '${ctx}/cell/param/toCellDetailParamInfoPage.action?smallCellCode='+small_cell_code+'&connection_status=' + status;

            enbvm.infoslide.url = url;
            enbvm.$refs.info.showSlide({timeZone: timeZone}, function(){
                $("#tabAlarmCli").siblings("li").removeClass("active");  
                $("#tabAlarmCli").siblings("li").eq(0).addClass("active");
                var thisPar = $("#tabAlarmCli").closest('.panelDefault');
                var tabClass = $("#tabAlarmCli").siblings("li").eq(0).attr('tabtit');
                $("." + tabClass).show().siblings("div").hide();
                
                if(thisPar.find("div").hasClass('omcTabsPage_second')){
                    $("." + tabClass).find(".omcPageTitleContainer_second > li").first().click();
                }
                try{
                    setTimeout(function() {
                        window.dispatchEvent(new Event('resize'));
                    },200);
                }catch(e){}
            });

            return;
        }

        function openSyncDialog(cell_code) {
            
            enbvm.openSyncDialog = true;
            enbvm.isSyncloading = true;

            setTimeout(function() {
                $('#sync-content').html('');
                $('#sync-content').each(function(idx,item){
                    if($(item).is(':visible')) {
                        $(item).load('${ctx}/cell/param/toSyncParamsPage.action',function(html) {
                            enbvm.isSyncloading = false;
                        })
                    }
                })
            },200);
            
        }

        /**
        * 下发参数查询，刷新小站信息
        * @param cell_code{string}: 基站编码
        **/
        function refreshCell(cell_code) {
            var selCell = enbvm.selectedRow;

            if (!selCell) {
                $.messager.alert(TiShi, "Error.");
                return;
            }
            
            selCell.connection_status = 'updating';

            var params = {smallCellCode: cell_code};
            
            $.post( "${ctx}/cell/param/refreshCellInfo.action", params, function(data){
                if (data["success"]) {
                } else {
                    showMsg('error_msg',data["message"]);
                }
            }, "json");
        }
        function ipAddrFormatter(value,row,index) {
            if(value) {
                value = '<a href="https://' + value + '" target="_blank" style="color: #4d84ff;">'+value+'</a>';
            }

            return value;
        }

        function cellStateFormatter(value, rowData, rowIndex){
            if (value == null) {
                return null;
            }
            else if (value == "1") {
                val = "<%= rb.getString("JiHuo")%>";
                value = "<div class='activeStatusItem'>"+(val)+"</div>"
            }
            else if (value == "0") {
                //状态不一样展示的文字也不一样
                val = "<%= rb.getString("QuJiHuo")%>";
                value = "<div class='inactiveStatusItem'>"+(val)+"</div>"
            }
            return value;
        }
        /**
        * 判断对象是否为空
        * @param obj{object}: 要判断的对象
        **/
        function isEmptyObject(obj){
            for(var key in obj){
                return false;
            }
            return true;
        }

        function halobStatusFormatter (value, rowData, rowIndex) {
            if ("1" == value) {
                value = "<span class='el-icon el-icon-status-enable' style='margin-right:10px;font-size: 20px;'></span>"+Kai;
            } else if ("0" == value) {
                value = "<span class='el-icon el-icon-status-disable' style='margin-right:10px;font-size: 20px;'></span>"+Guan;
            } else {
                value = "--";
            }
            return value;
        }

        function kpiStatusFormatter(value,row,index){
            if(value == null){
                return null;
            }else if(value == "off"){
                value = "<span class='el-icon el-icon-status-kpi-off' style='float:left;font-size:20px;margin-right:10px'></span>"+"<span style='color:#444;margin-top:4px;display:block;float:left'>"+(value)+"</span>"		
            }else if(value == "normal"){
                value = "<span class='el-icon el-icon-status-kpi-normal' style='float:left;font-size:20px;margin-right:10px'></span>"+"<span style='color:#444;margin-top:4px;display:block;float:left'>"+(value)+"</span>"
            }else if(value == "broken"){
                value = "<span class='el-icon el-icon-status-kpi-failure' style='float:left;font-size:20px;margin-right:10px'></span>"+"<span style='color:#444;margin-top:4px;display:block;float:left'>"+(value)+"</span>"
            }
            return value;
        }

        function syncStatusFormatter(value, rowData, rowIndex){
            if (value == null) {
                return null;
            } else if (value == "--" ){
                return "--";
            } else if (value == ("GPS " +"<%= rb.getString("ZhengZaiTongBu")%>")) {
                val ="GPS "+ "<%= rb.getString("ZhengZaiTongBu")%>";
                value = "<img style='margin:0px 10px 0px 0;float:left;width: 18px;' src ='${ctx}/css/images/main/monitor-ico/monitor-syning.gif'/>"+"<span style='color:#444;display:block;float:left'>"+(val)+"</span>"
            }else if (value == ("1588 " +"<%= rb.getString("ZhengZaiTongBu")%>")) {
                val ="1588 "+ "<%= rb.getString("ZhengZaiTongBu")%>";
                value = "<img style='margin:0px 10px 0px 0;float:left;width: 18px;' src ='${ctx}/css/images/main/monitor-ico/monitor-syning.gif'/>"+"<span style='color:#444;display:block;float:left'>"+(val)+"</span>"
            }else if (value == ("REM " +"<%= rb.getString("ZhengZaiTongBu")%>")) {
                val ="REM "+ "<%= rb.getString("ZhengZaiTongBu")%>";
                value = "<img style='margin:0px 10px 0px 0;float:left;width: 18px;' src ='${ctx}/css/images/main/monitor-ico/monitor-syning.gif'/>"+"<span style='color:#444;display:block;float:left'>"+(val)+"</span>"
            }else if (value == "GPS "+ "<%= rb.getString("TongBuChengGong")%>" ) {
                val ="GPS "+ "<%= rb.getString("TongBuChengGong")%>";
                value = "<img style='margin:0px 10px 0px 0;float:left;width: 18px;' src ='${ctx}/css/images/main/monitor-ico/tnsuccessomc.png'/>"+"<span style='color:#444;display:block;float:left'>"+(val)+"</span>"
            }else if (value == "1588 "+"<%= rb.getString("TongBuChengGong")%>" ) {	
                val = "<%= rb.getString("TongBuChengGong")%>";
                val = "1588 " + val;
                value = "<img style='margin:0px 10px 0px 0;float:left;width: 18px;' src ='${ctx}/css/images/main/monitor-ico/tnsuccessomc.png'/>"+"<span style='color:#444;display:block;float:left'>"+(val)+"</span>"
            }else if (value == "REM "+"<%= rb.getString("TongBuChengGong")%>") {	
                val = "<%= rb.getString("TongBuChengGong")%>";
                val = "REM " + val;
                value = "<img style='margin:0px 10px 0px 0;float:left;width: 18px;' src ='${ctx}/css/images/main/monitor-ico/tnsuccessomc.png'/>"+"<span style='color:#444;display:block;float:left'>"+(val)+"</span>"
            }else if (value == "<%= rb.getString("WeiTongBu")%>") {	
                val = "<%= rb.getString("WeiTongBu")%>";
                value = "<img style='margin:0px 10px 0px 0;float:left;width: 18px;' src ='${ctx}/css/images/main/monitor-ico/monitor-nosynomc.png'/>"+"<span style='color:#444;display:block;float:left'>"+(val)+"</span>"
            }
            return '<div style="min-width: 130px;diplay: line-block;">'+value+'</div>';
        }

        function lockStatusFmt(value,rowData,rowIndex){
            if(value == "--"){
                return value;
            }else{
                if(value == 0){
                    return "<div><span class='el-icon el-icon-status-unlock' style='margin-right:10px;font-size:20px;'></span><%=rb.getString("YouXiaoQiJieSuo")%></div>";
                }else{
                    return "<div><span class='el-icon el-icon-operation-lock yellowIcon' style='margin-right:10px;font-size:20px;'></span><%=rb.getString("YouXiaoQiSuoDing")%></div>";
                }
            }
        }

        function confirmImmediateCollectLogFile(divId) {
            var selCell = enbvm.selectedRow,
                cellCode = selCell.small_cell_code,
                serial_number = selCell.serial_number,
                param = {
                    start_time: 'undefined',
                    end_time: 'undefined',
                    execute_type: 'Immediately',
                    reportPeriod: '',
                    isReboot: 'false',
                    serial_number: serial_number, 				    				
                    timeZone: timeZone, 
                    device_type: "eNB",
                    device_code: cellCode,
                    logType: 'deviceLog'
                };
            
            $.post("${ctx}/cell/collect/goImmediateCollectLogFile.action", param, function (data) {
                if (data["success"]) {
                    showMsg('success_msg','<%=rb.getString("RiZhiZhengZaiShouJi")%>')
                } else {
                    showMsg('error_msg',data['message']);
                }
            }, "json");
        }

        // 发起SAS注册
        function registerSas() {
            var selCell = enbvm.selectedRow;

            $.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingFaQiSASZhuCe")%>" , function(r) {
                if (r) {
                    var params = {
                        cellCode: selCell["small_cell_code"],
                        serialNumber: selCell["serial_number"]
                    }

                    $.post("${ctx}/cell/SAS/registerCellForSas.action", params, function(data){
                        if (!data["success"]) {
                            showMsg('error_msg',data["message"]);
                        }
                    }, "json");
                }
            });
        }
        // 取消sas注册
        function deregisterSAS(){
            var selCell = enbvm.selectedRow;
            var params = {
                "sn": selCell["serial_number"],
                "enbCode": selCell["small_cell_code"]
            }
            
            $.messager.confirm("<%=rb.getString("QueRen")%>", "Are you sure to deregister?" , function(r) {
                if (r) {
                    $.post("${ctx}/cell/SAS/deregister.action", params, function(data){
                        if (!data["success"]) {
                            showMsg('error_msg',data["message"]);
                        }
                    }, "json");
                }
            });
        }

        function cellReboot(cell_code, product) {
            var selCell = enbvm.selectedRow;

            $.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingChongQiSheBei")%>", function (r) {
                if (r) {
                    showMsg('prompt_msg',"<%=rb.getString("MingLingYiXiaFa")%>")
                    var param = {
                        cell_code: cell_code
                    };
                    $.post("${ctx}/cell/cpeinfos/cellReboot.action", param, function (data) {
                        if (!data["success"]) {
                            showMsg('error_msg', data["message"]);
                        }
                    }, "json");
                }
            }).addClass("seriousConfirm");
        }

        function openCloseHalob(cellCode,halobSwitch){
            var params = {};
            params.cell_code = cellCode;
            params.halob_switch = halobSwitch;
            $.messager.confirm({
                title: '<%=rb.getString("TiShi")%>',
                msg: '<%=rb.getString("CanShuXiuGaiXuYaoChongQiJiZhan")%>',
                fn:function(r){
                    if(r){
                        $.post("${ctx}/cell/cpeinfos/setCellHalobSwitch.action",params,function(data){
                            if(data["success"]){
                                /* $('#tableHomeCellList').datagrid('reload'); */
                            }
                        },"json")
                    }
                }
            }).addClass("seriousConfirm");
            
        }

        function activeOpStatus(active,smallCellCode,cellNumber, type) {
            var vm = enbvm,
                params = {
                    op_state: active,
                    small_cell_code: smallCellCode
                },
                row = vm.selectedRow;
            
            if(cellNumber) params.cellNumber = cellNumber;
            if(vm.platformType == 'BM' && type == 'gsm') params.isGsm = '1';

            if(row.isSystemDeActivation == '1' && !['BM'].includes(vm.platformType)) { // 检测经纬度变化激活提示
                var confirmMsg = '<%=rb.getString("QuJiHuoQueRen")%>',
                    activeTips = '',
                    h = vm.$createElement;

                if(active == '1') {
                    confirmMsg = '<%=rb.getString("JiHuoQueRen")%>';
                    activeTips = '<%=rb.getString("WeiZhiJianCeBianHuaTi")%>';
                }

                vm.$msgbox({
                    title: '<%=rb.getString("QueRen")%>',
                    message: h('p', null, [
                        h('div', null, confirmMsg),
                        h('i', { style: 'color: #7A7992'}, activeTips)
                    ]),
                    showCancelButton: true,
                    customClass: 'warningConfirm',
                    confirmButtonText: '<%=rb.getString("QueDing")%>',
                    cancelButtonText: '<%=rb.getString("QuXiao")%>',
                    type: 'warning',
                    closeOnClickModal: false
                }).then((r)=>{
                    if(r == 'confirm') {
                        $.post("${ctx}/cell/cpeinfos/cellModifyActiveStatus.action",params,function(data){
                            if(data["success"]){
                                enbvm.refreshList();
                                showMsg('success_msg','<%=rb.getString("XiaFaChengGong")%>');
                            }else{
                                showMsg('error_msg',data["message"]);
                            }
                        },"json")
                    }
                })

            }else {
                $.post("${ctx}/cell/cpeinfos/cellModifyActiveStatus.action",params,function(data){
                    if(data["success"]){
                        enbvm.refreshList();
                        showMsg('success_msg','<%=rb.getString("XiaFaChengGong")%>');
                    }else{
                        showMsg('error_msg',data["message"]);
                    }
                },"json")

            }
        }

        function configReset(cellcode){
            $.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingHuiFuMoRenPeiZhi")%>", function (r) {
                if (r) {
                    var param = {
                            cellCode: cellcode
                    };
                    $.post("${ctx}/cell/cpeinfos/cellFactoryReset.action", param, function (data) {
                        
                    }, "json");
                }
            }).addClass("seriousConfirm");
        }
        //设置基站有效期
        function setEffectPeriod() {
            enbvm.goPeriod();
        }

        /*function setLockStatus(smallcellCode, lockStatus){
            var params = {
                    smallCellCode : smallcellCode
            }
            if(lockStatus == 0){
                params.lockStatus = 'lock';
            }else{
                params.lockStatus = 'unlock';
            }
            $.post("${ctx}/cell/cpeinfos/operationLockStatus.action",params,function(data){
                if(data["success"]){
                    enbvm.refreshList();
                }else{
                    showMsg('error_msg',data["message"]);
                }
            },"json")
        }*/

        function setForceRFStatus(code, status){
            $.ajax({
                url: '${ctx}/cell/SAS/autoOrForceRF.action',
                data: {serialNumber: code, autoOrForceRF: status},
                type: 'post',
                dataType: 'json',
                success: function(data){
                    if(data.success){
                        enbvm.refreshList();
                    }
                    toast(data.message,$('#omc_app_ctn'),'success');
                },
                error: function(data){
                    toast('<%=rb.getString("SheZhiShiBai")%>',$('#omc_app_ctn'));
                }
            });
        }

        function distributeSn(divId,small_cell_code){
            enbvm.goDistribute(small_cell_code)
        }
        function goSettingPanel(small_cell_code, sn, connStatus, carrierType){
        	enbvm.goSettingSlide(small_cell_code, sn, connStatus, carrierType)
        }

        function goCellDetailAlarmInfoWin(row) {
            var rowDatas = row,
                platformType = rowDatas.platformType,
                dualCarrierType = rowDatas.dual_carrier_type,
                connectStatus = rowDatas.connection_status,
                idval = rowDatas.small_cell_code+"",
                id4param = idval;
            // 新类型QA_436Q_DC的处理
            if(['QA_436Q_CA','QA_436Q_SC','QA_436Q_DC','NEU430_DC'].includes(platformType) && dualCarrierType == 2) {// 辅波查看不可操作，更多操作只有射频
                id4param = id4param.substr(0,id4param.length-2);
            }
            enbvm.selectedRow = rowDatas;
             //判断菜单的位置
            $.ajax({
                url: '${ctx}/cell/quicksettings/getSettingGroupTree.action',
                dataType: 'json',
                async: false,
                type: 'post',
                data: {
                    title:'Settings',
                    smallCellCode: id4param
                },
                success: function(json){
                    if(json){
                        // 清空form元素
                        $('#enbSetting_slide_body').html('');
                        Render.tableCollector = {};
                        // 导航容器
                        var navCtn = $('#quick_setting_nav');
                        navCtn.empty();
                        json.map(function(item,idx){
                            var disabled = false;
                            if(item.code == "Basic"){
                                disabled = false;
                            }else{
                                if(connectStatus == 'Off') disabled = true;
                                
                                if(['QA_436Q_DC','NEU430_DC','Intel_CR_DC','MLN_DC'].includes(platformType) && dualCarrierType == 2) { // 新类型QA_436Q_DC的处理，辅波只有basic可以设置
                                    
                                    if(('Intel_CR_DC' == platformType || 'MLN_DC' == platformType ) && item.text == 'LTE') {// 4860 LTE 放开设置
                                        
                                    }else {
                                        disabled = true;
                                    }
                                }
                                
                                if(['QA_436Q_DC'].includes(platformType) && dualCarrierType == 2 && item.text == 'LTE') {
                                    disabled = false;
                                }
                            }
                            var iTitle = item.text;
                            var mItem = {platform: rowDatas.platform_flag,id: item.code, code: item.id,text: iTitle,disable:disabled, subTitle: '(<%=rb.getString("XiaoZhanBianMa")%><%=rb.getString("MaoHao")%>'+rowDatas.serial_number+'<%=rb.getString("DouHao")%><%=rb.getString("HostName")%><%=rb.getString("MaoHao")%>'+(rowDatas.host_name||'')+')'};
                            //data[2].children.push(mItem);
                            // 生成Nav导航，绑定click事件
                            var navItem = $('<li>'+iTitle+'</li>');
                            
                            if(disabled) $(navItem).css({'cursor':'not-allowed',color: 'silver'});
                            if(!isLWAEnable && mItem.text=="LTE-TURBO")  $(navItem).css({'display':'none'});

                            navCtn.append(navItem);
                            navItem.on('click',function(){
                                if(disabled) {
                                    return false;
                                }
                                turnTabs(navItem);
                                if(mItem.text=="LTE-TURBO"){
                                    window.sessionStorage.setItem('apInfoSn',rowDatas.serial_number)
                                    var url = '${ctx}/cell/ap/toAPInfo.action'
                                    var params = {
                                        smallCellCode:rowDatas.small_cell_code,
                                        enbSerialNumber:rowDatas.serial_number,
                                        connectionStatus:rowDatas.connection_status
                                    }
                                    $('#enbSetting_slide_body').load(url,params,function(){
                                        var ctner = document.querySelector('#enbSetting_slide_body');
                                        var scripts = ctner.querySelectorAll('script');
                                        setTimeout(function() {
                                            Array.from(scripts).map(function(script) { /* 执行远程的脚本 */
                                                if (ctner.contains(script)) {
                                                    ctner.removeChild(script);
                                                }
                                                var newScript = document.createElement('script');
                                                newScript.type = 'text/javascript';
                                                newScript.innerHTML = script.innerHTML;
                                                ctner.appendChild(newScript);
                                            });
                                        }, 0);
                                        
                                        setTimeout(function() {
                                            try{
                                                $.parser.parse(ctner);
                                            }catch(e){}
                                        }, 0);
                                    });
                                }else if(["Special","特殊"].includes(mItem.text)) {
                                    var url = '${ctx}/cell/quicksettings/goSpecialParamPage.action',
                                        params = {
                                            smallCellCode: rowDatas.small_cell_code,
                                            enbSerialNumber: rowDatas.serial_number,
                                            connectionStatus: rowDatas.connection_status
                                        };

                                    enbvm.loadSpecialTab(url, params);
                                    $('#enbSetting_slide').data('params', {id: mItem.code, smallCellCode: rowDatas.small_cell_code});
                                }else{
                                    var params = Render.getFormDatas($('#enbSetting_slide_body'));
                                    var edit = !isEmptyJson(params)
                                    var addEdit = false
                                    if(addEdit){
                                        turnTabs(navItem);
                                        var rowCode = $('#enbSetting_slide').data('params').smallCellCode;
                                        settingTabClick(mItem.code,rowCode,mItem.text,mItem.subTitle,mItem.platform);
                                    }else{
                                        if(edit){// 参数有变动时，确认提示
                                            $.messager.confirm('Confirm','<%=rb.getString("QueDingLiKaiDangQianYeMian")%>',function(r){
                                                if(r){
                                                    turnTabs(navItem);
                                                    var rowCode = $('#enbSetting_slide').data('params').smallCellCode;
                                                    settingTabClick(mItem.code,rowCode,mItem.text,mItem.subTitle,mItem.platform);
                                                }
                                            });
                                        }else{// 无变动直接跳转
                                            turnTabs(navItem);
                                            var rowCode = $('#enbSetting_slide').data('params').smallCellCode;
                                            settingTabClick(mItem.code,rowCode,mItem.text,mItem.subTitle,mItem.platform);
                                        }
                                    }
                                
                                }
                            })
                        });

                    }else{
                        data[2].disable = true;
                    }
                }
            });
            goCellDetailParamInfoWin('',idval,connectStatus,'');
        }
        
        function setRFStatus(code,status,cell){
    		var rfStatus = status;
    		// 依据状态设置开、关
    		if(status == 'on') rfStatus = 'off';
    		if(status == 'off') rfStatus = 'on';

    		cell = cell || '';
    		$.ajax({
    			url: '${ctx}/cell/cpeinfos/cellModifyRadioStatus.action',
    			data: {smallCellCode: code, radioStatus: rfStatus,cellNumber:cell},
    			type: 'post',
    			dataType: 'json',
    			success: function(data){
                    if(data["success"]){
                        enbvm.refreshList();
                        showMsg('success_msg','<%=rb.getString("XiaFaChengGong")%>');
                    }else{
                        showMsg('error_msg', '<%=rb.getString("SheZhiShiBai")%>');
                    }
    			},
    			error: function(data){
    				showMsg('error_msg', '<%=rb.getString("SheZhiShiBai")%>');
    			}
    		});
    	}
      	//刷新基站监控 下 统计信息，填充状态栏
        function refresh_cellStatusStatistics(cb) {
            var params = {},dualStatus = '';
            
            $.extend(params, enbvm.queryParams);

            params["switch_status"] = dualStatus;
            params["isDual"] = true;
            params["isMonitor"] = true;
            
            $.post("${ctx}/cell/cpeinfos/getCellStatusStatistics.action", params, function(data) {
                if(!data["connection_status"]){
                    connectionStatus = "0/0";
                    connectionStatusRef = "0/0"
                }else{
                    connectionStatus = data["connection_status"];
                    connectionStatusRef = data["connection_status_ref"];
                    if($("#onlineStatusNum").prev().hasClass('greenType')){
                        $("#cellInfo #onlineStatusNum").text(data["connection_status"]);
                    }else if($("#onlineStatusNum").prev().hasClass('redType')){
                        $("#cellInfo #onlineStatusNum").text(data["connection_status_ref"]);
                    }
                }
                $('#online_count_rate').text("( "+connectionStatus+" )");
                if(!data["mme_status"]){
                    mmeStatus = "0/0";
                    mmeStatusRef = "0/0";
                    $("#cellInfo #mmeStatusNum").text("0/0");
                }else{
                    mmeStatus = data["mme_status"];
                    mmeStatusRef = data["mmeStatus_ref"];
                    if($("#mmeStatusNum").prev().hasClass('greenType')){
                        $("#cellInfo #mmeStatusNum").text(data["mme_status"]);
                    }else if($("#mmeStatusNum").prev().hasClass('redType')){
                        $("#cellInfo #mmeStatusNum").text(data["mmeStatus_ref"]);
                    }
                }
                $('#mme_count_rate').text("( "+mmeStatus+" )");
                if(!data["op_state"]){
                    opStateStatus = "0/0";
                    opStateStatusRef = "0/0";
                    $("#cellInfo #activeStatusNum").text("0/0");
                }else{
                    opStateStatus = data["op_state"];
                    opStateStatusRef = data["opState_ref"];
                    
                    if($("#activeStatusNum").prev().hasClass('greenType')){
                        $("#cellInfo #activeStatusNum").text(data["op_state"]);
                    }else if($("#activeStatusNum").prev().hasClass('redType')){
                        $("#cellInfo #activeStatusNum").text(data["opState_ref"]);
                    }
                }
                $('#active_count_rate').text("( "+opStateStatus+" )");
                if(cb && typeof cb == 'function') {
                    cb();
                }
            }, "json");
        }
    </script>
</body>
</html>