<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.el-input__inner[readonly] {
		background-color: #f5f7fa;
	}
	.el-select .el-input__inner[readonly] {
		background-color: #fff;
	}
	#cpeFactoryResetAddTask .alarmBottomLine{
		background-color:#E9E9E9;
		width: 100%;
		height: 1px;
		margin-bottom: 30px; 
	}
	#cpeFactoryResetAddTask .titleStyML{
		margin-left: 50px;
	}
	#cpeFactoryResetAddTask .deviceTableBox{
		margin:20px 0px 0px 76px;
		display: flex;
		height:372px;
		background:#FFFFFF;
	}
	#cpeFactoryResetAddTask .deviceSpecifiedBox{
		width: 200px;
		height: 370px;
		border:1px solid #E9E9E9;
		border-right: none;
	}
	#cpeFactoryResetAddTask .deviceSpecifiedTitle{
		height: 36px;
		width: 200px;
		font-size: 12px;
		line-height: 36px;
		text-align: center;
		background: #F6F7FB;
		box-sizing: border-box;
		border-bottom:1px solid #E9E9E9;
	}
	#cpeFactoryResetAddTask .specifiedTypeBox{
		flex: 1;
		padding-top: 30px;
		padding-left: 40px;
	}
	#cpeFactoryResetAddTask .specifiedTypeBox .el-radio__label{
		font-size: 12px !important;
	}
	#cpeFactoryResetAddTask .specifiedTypeBox .el-radio+.el-radio{
		margin-left: 0px;
		display: block;
	}
	
	#cpeFactoryResetAddTask .changeDeviceTableWarp{
		width: calc(100% - 200px)!important;
		font-size: 12px !important;
	}
	#cpeFactoryResetAddTask .changeDeviceTableWarp .pairgrid-right{
		top:40px!important;
		height: calc(100% - 40px)!important;
	}
	#cpeFactoryResetAddTask .changeDeviceTableWarp .el-pairgrid-title{
		top:15px!important;
		right: 15px!important;
	}
	#cpeFactoryResetAddTask .changeDeviceTableWarp .transition-box .el-form-item{
		display: inline-block;
		margin-right: 30px;
	}
	
	#cpeFactoryResetAddTask .el-radio__input.is-checked+.el-radio__label,
	#cpeFactoryResetAddTask .el-radio{
		color:#333333;
	}
	#cpeFactoryResetAddTask .footer{
		width:100%;
		border-top:1px solid #E9E9E9;
		position:absolute;
		bottom:0px;
		height:50px;
		line-height:50px;
		background:#FFFFFF;
		z-index:99;
	}
	#cpeFactoryResetAddTask .footer div{
		padding-left: 40px;
	}
	#cpeFactoryResetAddTask .el-form-item__error,
	.cpe-label-flex .el-form-item__error,
	.cpeIpModalWarp .el-form-item__error{
		padding-top: 0px;
	}
	#cpeFactoryResetAddTask .closeSlideBtn{
		position:absolute;
		top:10px;
		right:20px;
		overflow: hidden;
	}
	#cpeFactoryResetAddTask .mainWarp{
		overflow:hidden;
	}
	#cpeFactoryResetAddTask .basicBox{
		padding-top:40px;
	}
	#cpeFactoryResetAddTask .basicInfo{
		padding: 20px 76px 0;
	}
	#cpeFactoryResetAddTask .allSelectTable{
		border:1px solid #E9E9E9;
		margin-right:45px;
	}
	#cpeFactoryResetAddTask .testCpeCode{
		margin-left:76px;
		margin-top:6px;
	}
	
	
	/* 参数配置 */
    
    #cpeFactoryResetAddTask .nav-flex-item {
		flex: auto;
		overflow: auto;
	}
	#cpeFactoryResetAddTask .nav-tabs {
		border-right: 1px solid #eee;
		min-width: 138px;
	}
	#cpeFactoryResetAddTask .nav-tabs div {
		height: 36px;
		line-height: 36px;
		padding: 0 20px;
		cursor: pointer;
		border-bottom: 1px solid #eee;
		word-break: keep-all;
	}
	#cpeFactoryResetAddTask .nav-tabs div.active {
		color: #4D84FF;
		background-color: #EDF6FF;
	}
	#cpeFactoryResetAddTask .nav-title {
		padding: 0 20px;
		height: 40px;
		line-height: 40px;
		color: #363b4e;
		font-weight: bold;
		border-bottom: 1px solid #eee;
	}
	#cpeFactoryResetAddTask .nav-main {
		display: flex;
		flex-direction: column;
		flex: auto;
		overflow: auto;
	}
	#cpeFactoryResetAddTask .nav-content {
		padding: 30px 40px;;
	}
	#cpeFactoryResetAddTask .nav-content .el-form-item__label, .cpe-label-flex .el-form-item__label {
		text-align: left;
		line-height: 28px;
	}
	
	#cpeFactoryResetAddTask .formContentWarp .el-form-item {
		display: flex; 
		margin: 0 100px 22px 0;
	}
	
	#cpeFactoryResetAddTask .formWarp .titleWarp {
		display: flex;
		margin-bottom: 22px;
	}
	#cpeFactoryResetAddTask .formWarp .titleWarp span{
		font-size: 14px;
		color: #333333;
		font-weight: bold;
	}
	#cpeFactoryResetAddTask .formWarp .cpeCircleTip {
		width: 7px;
		height: 7px;
		background: #333;
		border-radius: 50%;
		margin: 5px 6px 0 0;
	}
	#cpeFactoryResetAddTask .formWarp .formContentWarp {
		display: flex;
		margin-left: 14px;
	}
	#cpeFactoryResetAddTask .formWarp .el-form-item .el-switch {
		margin-top: 4px;
	}
	#cpeFactoryResetAddTask .earfcnPciWarp{
		margin-left: 120px;
		width: 720px;
		height: auto;
		overflow: auto;
		margin-bottom:30px;
		display: flex;
	}

	#cpeFactoryResetAddTask .ipTable thead tr th:first-child .cell {
		display: none;
	}
	.cpeIpModalWarp .el-dialog__body {
		padding: 30px;
		position: relative;
		min-height: 130px;
	}
	.cpeIpModalWarp .el-dialog__body .el-form .el-form-item__label { 
		line-height: 28px;
	}
	.cpeAddErrorTip {
		color: #FA5555;
		font-size: 12px;
		margin-top: 5px;
	}
	.cpeIpModalWarp .editErrorTip {
		color: #FA5555;
		font-size: 12px;
		margin-left: 50px;
		margin-top: 4px;
	}
	.cpeIpModalWarp .el-form-item__error {
		left: 0;
	}
	.cpeIpAddressWarp {
		margin-left: 50px;
		height: auto;
		overflow: auto;
		margin-bottom:40px;
	}
	.cpeIpWarpFotter {
		position: absolute;
		bottom: 25px;
	}
	
	#cpeFactoryResetAddTask .iplistTitle {
		position: relative;
		height: 40px;
		line-height: 40px;
		margin-left: 14px;
	}
	#cpeFactoryResetAddTask .systemWarp .el-form-item .el-form-item__label {
		width: 200px;
		margin-right: 0;
		margin-left: 14px;
	}
	#cpeFactoryResetAddTask .systemWarp .el-form-item .el-form-item__content {
		margin-left: 200px;
	}
	
	#cpeFactoryResetAddTask .lteWidthWarp .el-form-item {
		margin-bottom: 22px;
	}
	#cpeFactoryResetAddTask .lteWidthWarp .el-form-item .el-form-item__label {
		width: 120px;
	}
	#cpeFactoryResetAddTask .lteWidthWarp .el-form-item .el-form-item__content {
		display: inline-block;
	}
	#cpeFactoryResetAddTask .earfcn-pci-line {
		display: flex;
		align-items: center;
	}

	#cpeFactoryResetAddTask .formContentWarp .el-form-item .el-form-item__content {
		margin-left: 30px;
	}
	#cpeFactoryResetAddTask .lanWarp .el-form-item .el-form-item__label {
		margin-right: 30px;
	}
	#cpeFactoryResetAddTask .systemWarp .el-form-item,
	#cpeFactoryResetAddTask .lanWarp .el-form-item {		
		margin-bottom: 22px;
	}
	#cpeFactoryResetAddTask .systemWarp .el-form-item__error{
		margin-left: 15px;
	}

	#cpeFactoryResetAddTask .el-switch.is-checked .el-switch__core{
		border-color: #4D84FF !important;
		background-color: #4D84FF !important;
	}
	
	#cpeFactoryResetAddTask .form-suffix,
	.cpeIpModalWarp .form-suffix { 
		border:1px solid #4D84FF; 
		display:inline-block;
		padding:0 10px;
		background:#F2F6FF;	
		width: auto;
	}
	#cpeFactoryResetAddTask .suffixItem,
	.cpeIpModalWarp .suffixItem { 		
		margin-right:10px;
		margin-bottom:10px 
	}
	#cpeFactoryResetAddTask .form-suffix .text,
	.cpeIpModalWarp .form-suffix .text { 
		font-size:12px;
		color:#333333;
		width: auto;
	}
	/*导入  */
	.cpeImportCard .w400{
		width:320px;
	}
	.cpeImportCard .el-form-item{
	 	margin-bottom:16px;
	}
	.cpeImportCard .el-dialog__body{
		padding:30px !important;
		background:#FFFFFF;
		border:none;
	}
	.cpeImportCard .el-form-item__label{
		line-height:26px;
	}
	.cpeImportCard .el-input__suffix{
		top:4px;
	}
	.cpeImportCard .fileAcceptTip{
		color:#999999;
		font-size:12px;
		margin-left:16px;
	}
	.cpeImportCard .el-icon-circle-info:before{
		color:#CFCFCF;
	}
	.cpeImportCard .el-dialog__footer{
		padding:20px 0 !important;
		text-align:left;
	}
	.cpeImportCard .el-dialog__footer .el-button:first-child,
	.cpe-label-flex .el-dialog__footer .el-button:first-child{
		margin:20px 0 0 30px;
	}
	.cpeImportCard .importFooter{
		border-top:1px solid #DCDFE6;	
	}
	.cpeImportCard .el-dialog__header .el-icon-close{
		top: 0px;
		font-size: 18px;
	}
	#cpeFactoryResetAddTask .selectScanMode .el-input__inner{
		height:28px !important;
		width:200px;
	}
	#cpeFactoryResetAddTask .selectOptions .el-input{
		height:28px !important;
	}
	#cpeFactoryResetAddTask .selectOptions .el-input__inner{
		height:28px !important;		
	}
	
	#cpeFactoryResetAddTask .paramsItemBoxCls{
		display: flex;
		align-items: center;
		margin-bottom: 20px;
		min-width: 740px;
	}
	#cpeFactoryResetAddTask .paramsItemBoxCls .el-form-item{
		margin-bottom: 0px;
	}
	#cpeFactoryResetAddTask .paramsItemLabelCls{
		width: 140px;
	}
	#cpeFactoryResetAddTask .exportBtnBoxCls{
		height: 28px;
		width: 130px;
		margin-left: 140px;
		border : 1px solid #E9E9E9;
		border-radius: 4px;
		font-size: 12px;
		display: flex;
		align-items: center;
		justify-content: center;
		cursor: pointer;
		margin-bottom: 20px;
	}
	#cpeFactoryResetAddTask .exportTemplateBoxCls{
		display: flex;
		align-items: center;
	}
	#cpeFactoryResetAddTask .exportTemplateTipCls{
		color: rgba(0,0,0,0.32);
		margin-right: 20px;
	}
	#cpeFactoryResetAddTask .el-collapse-item__header{
		border-bottom:1px solid #fff;
	}
	#cpeFactoryResetAddTask .el-collapse-item__arrow{
		position:absolute;
		left:20px;
		top:0px;
	}
	#cpeFactoryResetAddTask .el-collapse-item{
		position:relative;
		border-bottom:1px solid #E9E9E9;
	}
	#cpeFactoryResetAddTask .el-collapse-item__content{
		margin: 0px 40px;
		padding-bottom: unset;
	}
	#cpeFactoryResetAddTask .el-collapse-item__header .el-icon-arrow-right{
		font-size:16px;
	}
	#cpeFactoryResetAddTask .el-collapse-item__header .el-icon-arrow-right:before{
		content:"\e639";
		color:#BBB;
	}
	#cpeFactoryResetAddTask .el-collapse-item__header .is-active.el-icon-arrow-right:before{
		content:"\e638";
		color:#BBB;
	}
	#cpeFactoryResetAddTask .el-collapse-item__arrow.is-active{
		transform:rotate(0deg);
	}
	#cpeFactoryResetAddTask .el-collapse{
		border-top:1px solid #fff;
		border-bottom:1px solid #fff;
	}
	#cpeFactoryResetAddTask .el-collapse-item__wrap{
		border-bottom:1px solid #fff;
		padding-left: 0px;
	}
	#cpeFactoryResetAddTask .el-collapse-item__header{
		max-width:800px;
	}

	#cpeFactoryResetAddTask  .allowMoreInputBoxCls{
		position: relative;
		flex: 1;
	}
	#cpeFactoryResetAddTask .allowMoreInputBoxCls .allowMoreInputHeadCls{
		margin-bottom: 5px;
	}
	#cpeFactoryResetAddTask .allowMoreInputBoxCls .allowMoreInputHeadCls .allowMoreInputTitleCls{
		font-size: 14px;
		color: rgba(0, 0, 0, 0.8);
	}
	#cpeFactoryResetAddTask .allowMoreInputBoxCls .allowMoreInputHeadCls .allowMoreInputTipsCls{
		font-size: 14px;
		color: rgba(0, 0, 0, 0.32);
	}
	#cpeFactoryResetAddTask .allowMoreInputBoxCls .allowMoreInputContentCls{
		border: 1px solid #DFE2EE;
		width: 80%;
		min-width: 600px;
		min-height: 78px;
		padding: 10px;
		border-radius: 4px;
		box-sizing: border-box;
		position: relative;
	}
	#cpeFactoryResetAddTask .allowMoreInputBoxCls .allowMoreInputAddedCls{
		position: absolute;
		top: -23px;
		right: 0px;
	}
	#cpeFactoryResetAddTask .allowMoreInputBoxCls .allowMoreInputFieldCls{
		display: flex;
	}
	#cpeFactoryResetAddTask .allowMoreInputBoxCls .allowMoreInputFieldCls .el-input{
		width: 240px;
	}
	#cpeFactoryResetAddTask .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddBtnCls{
		height: 26px;
		width: 56px;
		display: flex;
		align-items: center;
		justify-content: center;
		color:var(--main-color);
		border: 1px solid var(--main-color);
		background:rgba(var(--main-color-rgba1),0.1);
		border-radius: 4px;
		box-sizing: border-box;
		margin-left: 10px;
		cursor: pointer;
	}
	#cpeFactoryResetAddTask .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddTipCls{
		margin-left: 10px;
	}
	#cpeFactoryResetAddTask .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddBtnCls .el-icon::before{
		font-size: 16px;
		color:var(--main-color);
	}
	#cpeFactoryResetAddTask .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddBtnCls span:nth-child(2){
		margin: 0px 3px;
	}
	#cpeFactoryResetAddTask .allowMoreInputBoxCls .allowMoreInputParamsCls{
		display: flex;
		flex-wrap: wrap;
		width: 100%;
		margin-top: 10px;
	}
	#cpeFactoryResetAddTask .allowMoreInputBoxCls .allowMoreInputParamsItemCls{
		height: 26px;
		display: flex;
		align-items: center;
		border: 1px solid #DFE2EE;
		border-radius: 4px;
		box-sizing: border-box;
		padding: 0px 10px;
		margin-right: 10px;
		margin-bottom: 5px;
		background: #F8F8FD;
	}
	#cpeFactoryResetAddTask .allowMoreInputBoxCls .allowMoreInputParamsItemCls span:nth-child(1){
		display: inline-block;
		max-width: 520px;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	#cpeFactoryResetAddTask .allowMoreInputBoxCls .allowMoreInputParamsItemCls span:nth-child(2){
		margin-left: 10px;
	}
	#cpeFactoryResetAddTask .allowMoreInputBoxCls .allowMoreInputParamsItemCls .el-icon-close{
		font-size: unset;
		position: unset;
		top: unset;
		right: unset;
	}
	#cpeFactoryResetAddTask .allowMoreInputBoxCls .allowMoreInputParamsItemCls .el-icon-close::before{
		font-size: 12px;
		color: #7A7992;
	}
	#cpeFactoryResetAddTask .allowMoreInputBoxCls .allowMoreInputFootCls{
		height: 18px;
	}
	#cpeFactoryResetAddTask .allowMoreInputBoxCls .allowMoreInputFootCls .inputErrorBoxCls{
		color:red;
		font-size:10px;
	}
	#nrLockAddDialog .half-item {
		display: flex;
		flex-wrap: wrap;
	}
	#nrLockAddDialog .half-item .el-form-item {
		flex: 1 1 40%;
		margin-right: 40px;
	}
</style>
<div class="flex-ctn" id="cpeFactoryResetAddTask" style='overflow:hidden;'>
	<div  id='temp_add_close' class="placeholder-bt closeSlideBtn circleIcon" placeholder="<%=rb.getString("GuanBi")%>">		
		<span class="el-icon el-icon-circle-close" @click="closePanel"></span>
	</div>
	<el-form ref="addform" :model="form" :rules="formRules" label-position="left" :hide-required-asterisk='true'>
		<div class="group-title not-extend titleStyML basicBox">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<!-- 基本信息 -->
		<div class="basicInfo">
			<el-form-item prop="taskName" label="<%=rb.getString("RenWuMingCheng")%>" label-width="120px" style="margin-bottom:20px;">
				<el-input style='width:400px;padding-top:7px;' v-model="form.taskName" :readonly="readonly" placeholder="<%=rb.getString("ZuiDaChangDu")%><%=rb.getString("MaoHao")%> 100"></el-input>
			</el-form-item>
		</div>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
		</div>
		<el-form-item prop='' style='margin-left:76px;margin-top:20px;' label='<%=rb.getString("SheBeiXingHao")%>' label-width="120px">
			<el-radio-group v-model="cpeModel" :disabled="readonly" @change="cpeModelChange">
				<el-radio label="4G" border size="small">4G CPE</el-radio>
				<el-radio label="5G" border size="small">5G CPE</el-radio>
			</el-radio-group>
		</el-form-item>
		<div class="deviceTableBox">
			<div class="deviceSpecifiedBox">
				<div class="deviceSpecifiedTitle"><%=rb.getString("ZhiDingSheBeiZhiXing")%></div>
				<div class="specifiedTypeBox">
					<el-radio-group v-model="selectAll" :disabled="readonly" @change="selectAllChange">
						<el-radio label="1" style="margin-bottom:26px;"><%=rb.getString("QuanBu")%></el-radio>
						<el-radio label="0"><%=rb.getString("ZhiDingZhiXing")%></el-radio>
					</el-radio-group>
				</div>
			</div>
			<div class="changeDeviceTableWarp">
				<el-pairgrid
					v-if="showPairGrid && selectAll == '0'" 
					:id="'select_device_list'" 
					:rownumber="true" 
					style="margin-right:45px;"
					ref="deviceListPairgrid" 
					:right-url="rightUrl" 
					:left-url="leftUrl" 
					:height="'100%'" 
					:row-key="'small_cell_code'" 
					:query-params="queryParams" 
					query-name="mac_address" 
					:title="deviceTitle" 
					:messages="{placeholder:'<%=rb.getString("CPEMacAddress")%>'}" 
					@selection-change="devicesChange" 
					@right-load-success="loadSuccessDevice">
					<template slot="left">
						<el-table-column type='selection' width="50" align="center"></el-table-column>	
						<el-table-column prop="connection_status" width="50">
							<template slot-scope="scope">
								<div :class="{
									'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
									'':scope.row.have_connected==2,
									'conn_exc':scope.row.connection_status=='Exception',
									'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
							</template>
						</el-table-column>				
						<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>" min-width="150"></el-table-column>
						<el-table-column prop="host_name" show-overflow-tooltip label="<%=rb.getString("CPEName")%>" min-width="250" ></el-table-column>
						<el-table-column prop="mac_address" show-overflow-tooltip label="<%=rb.getString("CPEMacAddress")%>" min-width="140"></el-table-column>
						<el-table-column prop="imsi" show-overflow-tooltip label="<%=rb.getString("IMSI")%>" min-width="140"></el-table-column>
						<el-table-column prop="pci" show-overflow-tooltip label="PCI" min-width="140"></el-table-column>
						<el-table-column prop="group_name" show-overflow-tooltip label="<%=rb.getString("SheBeiZu")%>" min-width="190" ></el-table-column>
					</template>
					<template slot='toolbar'>
						<el-query type="normal" @query="queryDeviceList" placeholder="<%=rb.getString("CPEBianMa")%> / <%=rb.getString("CPEName")%> / PCI"></el-query>
					</template>
					<template slot='right'>
						<el-table-column prop='host_name' label='<%=rb.getString("CPEName")%>'></el-table-column>
						<el-table-column prop="mac_address" show-overflow-tooltip label="<%=rb.getString("CPEMacAddress")%>" ></el-table-column>
					</template>
				</el-pairgrid>
				<el-ctable 
					id="all_device_list" 
					v-if="selectAll == '1'" 
					class="allSelectTable" 
					ref="all_device_list"  
					:url="leftUrl" 
					:height="height" 
					front-pagination="true" 
					pagination="true" 
					:query-params="queryParams">
				 	<template slot='toolbar'>
						<el-query type="normal" @query="queryDeviceList" placeholder="<%=rb.getString("CPEBianMa")%> / <%=rb.getString("CPEName")%> / PCI"></el-query>
					</template>
					<el-table-column prop="connection_status" width="50">
						<template slot-scope="scope">
							<div :class="{
								'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
								'':scope.row.have_connected==2,
								'conn_exc':scope.row.connection_status=='Exception',
								'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
						</template>
					</el-table-column>
					<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>" min-width="150"></el-table-column>
					<el-table-column prop="host_name" show-overflow-tooltip label="<%=rb.getString("CPEName")%>" min-width="250" ></el-table-column>
					<el-table-column prop="mac_address" show-overflow-tooltip label="<%=rb.getString("CPEMacAddress")%>" min-width="140"></el-table-column>
					<el-table-column prop="imsi" show-overflow-tooltip label="<%=rb.getString("IMSI")%>" min-width="140"></el-table-column>
					<el-table-column prop="pci" show-overflow-tooltip label="PCI" min-width="140"></el-table-column>
					<el-table-column prop="group_name" show-overflow-tooltip label="<%=rb.getString("SheBeiZu")%>" min-width="190" ></el-table-column>
				</el-ctable>
				
				<div v-if="!showPairGrid && selectAll == '0'">
					<el-ctable :id="'selected_device_list'" ref="selected_device_list"  :url="rightUrl" :height="height" front-pagination="true" pagination="true" :query-params="queryParams" class="allSelectTable">
						<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>" min-width="150"></el-table-column>
						<el-table-column prop="host_name" show-overflow-tooltip label="<%=rb.getString("CPEName")%>" min-width="250" ></el-table-column>
						<el-table-column prop="mac_address" show-overflow-tooltip label="<%=rb.getString("CPEMacAddress")%>" min-width="140"></el-table-column>
						<el-table-column prop="imsi" show-overflow-tooltip label="<%=rb.getString("IMSI")%>" min-width="140"></el-table-column>
						<el-table-column prop="group_name" show-overflow-tooltip label="<%=rb.getString("SheBeiZu")%>" min-width="190" ></el-table-column>
					</el-ctable>
				</div>
			</div>
		</div>
		<el-form-item class="testCpeCode" prop="cpeCodes"></el-form-item>

		<div class="alarmBottomLine"></div>
		
		<!-- Parameter Configuration -->
		<div style="height:30px; position: relative; display:flex;">
			<div class='commonFlex'>
				<div class="group-title not-extend titleStyML" >
					<span class="title-icon"></span>
					<span class="title-text"><%=rb.getString("CanShuZiPeiZhi")%></span>	
				</div>
				<!-- 执行修改后，OMC与 CPE交互较慢，导致用户造成错觉，页面提示 -->
				<div style='padding-top:8px;' class='commonNotes12'>
					<i class="el-icon el-icon-circle-info" style='margin-right: 6px;'></i><%=rb.getString("CaoZuoKeHuTiShi")%>
				</div>
			</div>
			<!-- 导入，导出 -->
			<div v-show="AddBtnShow" class="circleIcon placeholder-bt importBtn topBtn" placeholder="<%=rb.getString("DaoRu")%>" @click="importFileBtn" style="top: 0px; right: 86px;">		
				<span class="el-icon el-icon-circle-import"></span>
			</div>
			<div v-show="AddBtnShow" class="circleIcon placeholder-bt exportBtn topBtn" placeholder="<%=rb.getString("DaoChu")%>" @click="exportFileBtn" style="top: 0px; right: 36px;">			
				<span class="el-icon el-icon-circle-export"></span>
			</div>
		</div>
		
    	<div class="group" style="padding:20px 46px 90px 76px;" >
            <div style="width: 100%;border: 1px solid #e9e9e9; display: flex; flex-direction: row; flex: auto;">              
                <!-- 导航区域 -->
				<div class="nav-tabs">
					<div v-for="(item,index) in tabs"  :class="{active: activeCode == item.code}" @click="tabClick(item)">{{item.text}}</div>
				</div>
				<!-- 右侧详情区域 -->
				<div class="nav-main">
					<!-- 内容载入区域 -->
					<div class="nav-flex-item nav-content">
						<div class="formWarp">
							<!-- Basic -->
							<div v-show="activeCode=='basic'">
								<el-collapse v-model="activeCollapse">
									<el-collapse-item name="Mac">
										<template slot='title'>
											<p style="display:inline-block;margin-left:40px;">
												<span style="font-size:14px;font-weight:bold">MAC Filter</span>
											</p>
										</template>
										<div class="rightContentCls" >
											<div class="paramsItemBoxCls">
												<div class="paramsItemLabelCls">MAC Filter Enable</div>
												<el-form-item prop='macFilterEnable' style="width:40%;min-width:400px;" label="">
													<el-switch v-model="form.macFilterEnable" active-value="1" inactive-value="0"></el-switch>
												</el-form-item>
											</div>
											<div class="paramsItemBoxCls">
												<div class="paramsItemLabelCls">MAC Filter Mode</div>
												<el-form-item prop='macFilterMode' style="width:40%;min-width:400px;" label="">
													<el-radio-group v-model="form.macFilterMode" >
														<el-radio label="1" border size="small">Whitelist</el-radio>
														<el-radio label="0" border size="small">Blocklist</el-radio>
													</el-radio-group>
												</el-form-item>
											</div>
											<div class="paramsItemBoxCls">
												<div class="paramsItemLabelCls">MAC Add Mode</div>
												<el-form-item prop='' style="width:40%;min-width:400px;" label="">
													<el-radio-group v-model="form.macAddMode">
														<el-radio label="0" border size="small">Manual Input</el-radio>
														<el-radio label="1" border size="small">Batch Import</el-radio>
													</el-radio-group>
												</el-form-item>
											</div>
											<div class="paramsItemBoxCls" style="margin-bottom: 5px;">
												<div class="paramsItemLabelCls">MAC Address</div>
												<div class="allowMoreInputBoxCls">
													<div class="allowMoreInputContentCls">
														<div class="allowMoreInputAddedCls">
															Added (
															<span v-if="form.macFilterMode == '0'" style="color: #4D84FF;">{{form.macFilterBlackList.length}} / 128</span>
															<span v-if="form.macFilterMode == '1'" style="color: #4D84FF;">{{form.macFilterWhiteList.length}} / 128</span> )
														</div>
														<div class="allowMoreInputFieldCls">
															<el-input v-show="form.macAddMode == '0'" v-model="form.macAddress"></el-input>
															<el-upload ref="macAddressAddUpload"
																v-show="form.macAddMode == '1'"
																:before-upload='macAddressAddBeforeUpload' 
																:on-success='macAddressAddCheckFile' 
																:on-change="macAddressAddFileChange" 
																:show-file-list="false" 
																:action="macAddressAddUploadFileURL" 
																:data="macAddressAddFileParams" 
																name="macAddressUploadFile" 
																:auto-upload="false"
																accept=".xls,.xlsx">
																<el-input :readonly="true" :value="macAddressAddFileName" placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' style="width:300px;">
																	<a slot="append" class="el-icon el-icon-operation-import grayIcon" @click="macAddressAddFileSelect"></a>
																</el-input>
																<a slot="trigger" ref="macAddressAddFile_up"></a>
															</el-upload>
															<div v-show="form.macAddMode == '0'" class="allowMoreInputAddBtnCls" @click="macAddressAdd">
																<span class="el-icon el-icon-plus"></span>
																<span>Add</span>
															</div>
															<div class="allowMoreInputAddTipCls">
																<span style="color:red" v-if="macAddressErrorMessage">{{macAddressErrorMessage}}</span>
																<span style="color:rgba(0,0,0,0.32)" v-if="!macAddressErrorMessage && form.macAddMode == '0'">Format：xx:xx:xx:xx:xx:xx,no more than 128</span>
																<span style="color:rgba(0,0,0,0.32)" v-if="!macAddressErrorMessage && form.macAddMode == '1'"><%=rb.getString("DangQianZhiChiWenJianLeiXing")%></span>
															</div>
														</div>
														<div v-show="form.macAddMode == '1'" class="exportTemplateBoxCls">
															<div class="exportTemplateTipCls"><%=rb.getString("ShiYongMuBanDaoRuTiShi")%></div>
															<div @click="exportAddTemplate('MAC')" style="cursor:pointer;">
																<span class='el-icon el-icon-common-download exportTemplateIcon'></span>
																<span  class='exportTemplateText'><%=rb.getString("DaoChuMuBan")%></span>
															</div>
														</div>
														<div class="allowMoreInputParamsCls" v-show="form.macFilterMode == '1'">
															<div v-for="item in form.macFilterWhiteList" class="allowMoreInputParamsItemCls">
																<span>{{item.macAddress}}</span>
																<span class="el-icon el-icon-close" @click="macAddressListDel(item)"></span>
															</div>
														</div>
														<div class="allowMoreInputParamsCls" v-show="form.macFilterMode == '0'">
															<div v-for="item in form.macFilterBlackList" class="allowMoreInputParamsItemCls">
																<span>{{item.macAddress}}</span>
																<span class="el-icon el-icon-close" @click="macAddressListDel(item)"></span>
															</div>
														</div>
														<el-form-item prop='macFilterWhiteList' style="display:none;" label="" label-width="0px">
															<el-input v-model='form.macFilterWhiteList'></el-input>
														</el-form-item>
														<el-form-item prop='macFilterBlackList' style="display:none;" label="" label-width="0px">
															<el-input v-model='form.macFilterBlackList'></el-input>
														</el-form-item>
													</div>
												</div>
											</div>
											<div class="exportBtnBoxCls" @click="macAddressListExportClick">
												<span class="el-icon el-icon-operation-export"></span>
												<span style="margin-left: 5px;">Export MAC List</span>
											</div>
										</div>
									</el-collapse-item>
									<el-collapse-item name="IP">
										<template slot='title'>
											<p style="display:inline-block;margin-left:40px;">
												<span style="font-size:14px;font-weight:bold">IP Filter</span>
											</p>
										</template>
										<div class="rightContentCls" >
											<div class="paramsItemBoxCls">
												<div class="paramsItemLabelCls">IP Filter Enable</div>
												<el-form-item prop='ipFilterEnable' style="width:40%;min-width:400px;" label="">
													<el-switch v-model="form.ipFilterEnable" active-value="1" inactive-value="0"></el-switch>
												</el-form-item>
											</div>
											<div class="paramsItemBoxCls">
												<div class="paramsItemLabelCls">IP Filter Mode</div>
												<el-form-item prop='ipFilterMode' style="width:40%;min-width:400px;" label="">
													<el-radio-group v-model="form.ipFilterMode" >
														<el-radio label="1" border size="small">Whitelist</el-radio>
														<el-radio label="0" border size="small">Blocklist</el-radio>
													</el-radio-group>
												</el-form-item>
											</div>
											<div class="paramsItemBoxCls">
												<div class="paramsItemLabelCls">IP Add Mode</div>
												<el-form-item prop='' style="width:40%;min-width:400px;" label="">
													<el-radio-group v-model="form.ipAddMode">
														<el-radio label="0" border size="small">Manual Input</el-radio>
														<el-radio label="1" border size="small">Batch Import</el-radio>
													</el-radio-group>
												</el-form-item>
											</div>
											<div class="paramsItemBoxCls" style="margin-bottom: 5px;">
												<div class="paramsItemLabelCls">Source IP Address</div>
												<div class="allowMoreInputBoxCls">
													<div class="allowMoreInputContentCls">
														<div class="allowMoreInputAddedCls">
															Added (
															<span v-if="form.ipFilterMode == '0'" style="color: #4D84FF;">{{form.ipFilterBlackList.length}} / 128</span>
															<span v-if="form.ipFilterMode == '1'" style="color: #4D84FF;">{{form.ipFilterWhiteList.length}} / 128</span> )
														</div>
														<div class="allowMoreInputFieldCls">
															<el-input v-show="form.ipAddMode == '0'" v-model="form.sourceIp"></el-input>
															<el-upload ref="sourceIpAddUpload"
																v-show="form.ipAddMode == '1'"
																:before-upload='sourceIpAddBeforeUpload' 
																:on-success='sourceIpAddCheckFile' 
																:on-change="sourceIpAddFileChange" 
																:show-file-list="false" 
																:action="sourceIpAddUploadFileURL" 
																:data="sourceIpAddFileParams" 
																name="sourceIpUploadFile" 
																:auto-upload="false"
																accept=".xlsx, .csv">
																<el-input :readonly="true" :value="sourceIpAddFileName" placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' style="width:300px;">
																	<a slot="append" class="el-icon el-icon-operation-import grayIcon" @click="sourceIpAddFileSelect"></a>
																</el-input>
																<a slot="trigger" ref="sourceIpAddFile_up"></a>
															</el-upload>
															<div v-show="form.ipAddMode == '0'" class="allowMoreInputAddBtnCls" @click="sourceIpAdd">
																<span class="el-icon el-icon-plus"></span>
																<span>Add</span>
															</div>
															<div class="allowMoreInputAddTipCls">
																<span style="color:red" v-if="sourceIpErrorMessage">{{sourceIpErrorMessage}}</span>
																<span style="color:rgba(0,0,0,0.32)" v-if="!sourceIpErrorMessage && form.ipAddMode == '0'">Support IPv4 or IPv6,no more than 128</span>
																<span style="color:rgba(0,0,0,0.32)" v-if="!sourceIpErrorMessage && form.ipAddMode == '1'"><%=rb.getString("DangQianZhiChiWenJianLeiXing")%></span>
															</div>
														</div>
														<div v-show="form.ipAddMode == '1'" class="exportTemplateBoxCls">
															<div class="exportTemplateTipCls"><%=rb.getString("ShiYongMuBanDaoRuTiShi")%></div>
															<div @click="exportAddTemplate('IP')" style="cursor:pointer;">
																<span class='el-icon el-icon-common-download exportTemplateIcon'></span>
																<span  class='exportTemplateText'><%=rb.getString("DaoChuMuBan")%></span>
															</div>
														</div>
														<div class="allowMoreInputParamsCls" v-show="form.ipFilterMode == '1'">
															<div v-for="item in form.ipFilterWhiteList" class="allowMoreInputParamsItemCls">
																<span>{{item.sourceIp}}</span>
																<span class="el-icon el-icon-close" @click="sourceIpListDel(item)"></span>
															</div>
														</div>
														<div class="allowMoreInputParamsCls" v-show="form.ipFilterMode == '0'">
															<div v-for="item in form.ipFilterBlackList" class="allowMoreInputParamsItemCls">
																<span>{{item.sourceIp}}</span>
																<span class="el-icon el-icon-close" @click="sourceIpListDel(item)"></span>
															</div>
														</div>
														<el-form-item prop='ipFilterWhiteList' style="display:none;" label="" label-width="0px">
															<el-input v-model='form.ipFilterWhiteList'></el-input>
														</el-form-item>
														<el-form-item prop='ipFilterBlackList' style="display:none;" label="" label-width="0px">
															<el-input v-model='form.ipFilterBlackList'></el-input>
														</el-form-item>
													</div>
												</div>
											</div>
											<div class="exportBtnBoxCls" @click="sourceIpListExportClick">
												<span class="el-icon el-icon-operation-export"></span>
												<span style="margin-left: 5px;">Export IP List</span>
											</div>
										</div>
									</el-collapse-item>
									<el-collapse-item name="URL">
										<template slot='title'>
											<p style="display:inline-block;margin-left:40px;">
												<span style="font-size:14px;font-weight:bold">URL Blocklist Filter</span>
											</p>
										</template>
										<div class="rightContentCls" >
											<div class="paramsItemBoxCls">
												<div class="paramsItemLabelCls">URL Filter Enable</div>
												<el-form-item prop='urlFilterEnable' style="width:40%;min-width:400px;" label="">
													<el-switch v-model="form.urlFilterEnable" active-value="1" inactive-value="0"></el-switch>
												</el-form-item>
											</div>
											<div class="paramsItemBoxCls">
												<div class="paramsItemLabelCls">URL Add Mode</div>
												<el-form-item prop='' style="width:40%;min-width:400px;" label="">
													<el-radio-group v-model="form.urlAddMode">
														<el-radio label="0" border size="small">Manual Input</el-radio>
														<el-radio label="1" border size="small">Batch Import</el-radio>
													</el-radio-group>
												</el-form-item>
											</div>
											<div class="paramsItemBoxCls" style="margin-bottom: 5px;">
												<div class="paramsItemLabelCls">URL</div>
												<div class="allowMoreInputBoxCls">
													<div class="allowMoreInputContentCls">
														<div class="allowMoreInputAddedCls">
															Added (
															<span style="color: #4D84FF;">{{form.urlFilterBlackList.length}} / 64</span> )
														</div>
														<div class="allowMoreInputFieldCls">
															<el-input v-show="form.urlAddMode == '0'" v-model="form.url"></el-input>
															<el-upload ref="urlAddUpload"
																v-show="form.urlAddMode == '1'"
																:before-upload='urlAddBeforeUpload' 
																:on-success='urlAddCheckFile' 
																:on-change="urlAddFileChange" 
																:show-file-list="false" 
																:action="urlAddUploadFileURL" 
																:data="urlAddFileParams" 
																name="urlUploadFile" 
																:auto-upload="false"
																accept=".xlsx, .xls">
																<el-input :readonly="true" :value="urlAddFileName" placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' style="width:300px;">
																	<a slot="append" class="el-icon el-icon-operation-import grayIcon" @click="urlAddFileSelect"></a>
																</el-input>
																<a slot="trigger" ref="urlAddFile_up"></a>
															</el-upload>
															<div v-show="form.urlAddMode == '0'" class="allowMoreInputAddBtnCls" @click="urlAdd">
																<span class="el-icon el-icon-plus"></span>
																<span>Add</span>
															</div>
															<div class="allowMoreInputAddTipCls">
																<span style="color:red" v-if="urlErrorMessage">{{urlErrorMessage}}</span>
																<span style="color:rgba(0,0,0,0.32)" v-if="!urlErrorMessage && form.urlAddMode == '0'">no more than 64</span>
																<span style="color:rgba(0,0,0,0.32)" v-if="!urlErrorMessage && form.urlAddMode == '1'"><%=rb.getString("DangQianZhiChiWenJianLeiXing")%></span>
															</div>
														</div>
														<div v-show="form.urlAddMode == '1'" class="exportTemplateBoxCls">
															<div class="exportTemplateTipCls"><%=rb.getString("ShiYongMuBanDaoRuTiShi")%></div>
															<div @click="exportAddTemplate('URL')" style="cursor:pointer;">
																<span class='el-icon el-icon-common-download exportTemplateIcon'></span>
																<span  class='exportTemplateText'><%=rb.getString("DaoChuMuBan")%></span>
															</div>
														</div>
														<div class="allowMoreInputParamsCls">
															<div v-for="item in form.urlFilterBlackList" class="allowMoreInputParamsItemCls">
																<span>{{item.url}}</span>
																<span class="el-icon el-icon-close" @click="urlListDel(item)"></span>
															</div>
														</div>
														<el-form-item prop='urlFilterBlackList' style="display:none;" label="" label-width="0px">
															<el-input v-model='form.urlFilterBlackList'></el-input>
														</el-form-item>
													</div>
												</div>
											</div>
											<div class="exportBtnBoxCls" @click="urlListExportClick">
												<span class="el-icon el-icon-operation-export"></span>
												<span style="margin-left: 5px;">Export URL List</span>
											</div>
										</div>
									</el-collapse-item>
								</el-collapse>
							</div>
							<!-- Network -->
							<div v-show="activeCode=='network'">
								<!-- WLAN  -->
			 					<div>
				 					<div class="titleWarp">
				 						<div class="cpeCircleTip"></div>
				 						<span>WLAN</span>
				 					</div>
				 					<div style="display: none;">
										<el-form-item prop="wifiSsid"></el-form-item>

										<el-form-item prop="wifiEncryption"></el-form-item>
										<el-form-item prop="wifiPassphrase"></el-form-item>
			
										<el-form-item prop="wifi1Enable"></el-form-item>
										<el-form-item prop="wifi1Ssid"></el-form-item>

										<el-form-item prop="wifi1Encryption"></el-form-item>
										<el-form-item prop="wifi1Passphrase"></el-form-item>
			
										<el-form-item prop="wifi2Enable"></el-form-item>
										<el-form-item prop="wifi2Ssid"></el-form-item>

										<el-form-item prop="wifi2Encryption"></el-form-item>
										<el-form-item prop="wifi2Passphrase"></el-form-item>
			
										<el-form-item prop="wifi3Enable"></el-form-item>
										<el-form-item prop="wifi3Ssid"></el-form-item>

										<el-form-item prop="wifi3Encryption"></el-form-item>
										<el-form-item prop="wifi3Passphrase"></el-form-item>
									</div>	 					
				 					<div class="formContentWarp" >
				 						<el-form-item label="WiFi" prop="wlanEnable">
											<el-select v-model="form.wlanEnable" :disabled="readonly" class="selectOptions" @change="wlanEnableChange"> 
												<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label="Frequency(Channel)" prop="wifiChannel">
											<el-select v-model="form.wifiChannel" :disabled="readonly" size="mini" :readonly="readonly">
												<el-option label="" value=""></el-option>
												<el-option label="AUTO" value="AUTO"></el-option>
												<el-option label="2.412GHz(Channel 1)" value="1"></el-option>
												<el-option label="2.417GHz(Channel 2)" value="2"></el-option>
												<el-option label="2.422GHz(Channel 3)" value="3"></el-option>
												<el-option label="2.427GHz(Channel 4)" value="4"></el-option>
												<el-option label="2.43GHz(Channel 5)" value="5"></el-option>
												<el-option label="2.437GHz(Channel 6)" value="6"></el-option>
												<el-option label="2.442GHz(Channel 7)" value="7"></el-option>
												<el-option label="2.447GHz(Channel 8)" value="8"></el-option>
												<el-option label="2.452GHz(Channel 9)" value="9"></el-option>
												<el-option label="2.457GHz(Channel 10)" value="10"></el-option>
												<el-option label="2.462GHz(Channel 11)" value="11"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label="Channel Bandwidth" prop="wifiBandwidth">
											<el-radio-group v-model="form.wifiBandwidth" style="margin-top: 6px;" :disabled="isWifiClosed && readonly">
												<el-radio label="0">20M</el-radio>
												<el-radio label="1" style="margin-left: 30px;">20/40M</el-radio>
											</el-radio-group>
										</el-form-item>
				 					</div>
				 					
				 					<div style="margin-left: 14px; margin-bottom: 22px;">		 					
				 						<div style="margin-bottom: 10px; font-size: 14px; ">MBSSID</div>
										<el-ctable height="177" ref="taskList" :data="tbData" :pagination="false" style="border: 1px solid #E9E9E9;width:80%; ">
											<el-table-column width="80">
												<template slot-scope="scope">
													<i class="el-icon el-icon-operation-edit" @click="modifyWifi(scope.row)" v-show="!isWifiClosed && editBtnShow"></i>
												</template>
											</el-table-column>
											<el-table-column label="Network Name(SSID)" prop="wifiSsid"></el-table-column>
											<el-table-column label="Security Mode" prop="wifiEncryption">
												<template slot-scope="scope">
													{{modeKeys[scope.row.wifiEncryption]}}
												</template>
											</el-table-column>
											<el-table-column label="Status" prop="wifiEnable">
												<template slot-scope="scope">
													<span v-if="scope.row.wifiEnable=='1'">Enable</span>
													<span v-if="scope.row.wifiEnable=='0'">Disable</span>
												</template>
											</el-table-column>									
										</el-ctable>
				 					</div>
			 					
			 						<!-- wifi5 -->
			 						<div style="display: none;">
										<el-form-item prop="wifi5WifiSsid"></el-form-item>
										<el-form-item prop="wifi5WifiEncryption"></el-form-item>
										<el-form-item prop="wifi5WifiPassphrase"></el-form-item>
			
										<el-form-item prop="wifi5Wifi1Enable"></el-form-item>
										<el-form-item prop="wifi5Wifi1Ssid"></el-form-item>
										<el-form-item prop="wifi5Wifi1Encryption"></el-form-item>
										<el-form-item prop="wifi5Wifi1Passphrase"></el-form-item>
			
										<el-form-item prop="wifi5Wifi2Enable"></el-form-item>
										<el-form-item prop="wifi5Wifi2Ssid"></el-form-item>
										<el-form-item prop="wifi5Wifi2Encryption"></el-form-item>
										<el-form-item prop="wifi5Wifi2Passphrase"></el-form-item>
			
										<el-form-item prop="wifi5Wifi3Enable"></el-form-item>
										<el-form-item prop="wifi5Wifi3Ssid"></el-form-item>
										<el-form-item prop="wifi5Wifi3Encryption"></el-form-item>
										<el-form-item prop="wifi5Wifi3Passphrase"></el-form-item>
									</div>
			 						<div class="formContentWarp">
			 							<el-form-item label="WiFi5" prop="wifi5WlanEnable">
											<el-select v-model="form.wifi5WlanEnable" :disabled="readonly" class="selectOptions" @change="wifi5EnableChange"> 
												<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label="Network Mode" prop="wifi5WifiMode">
											<el-select v-model="form.wifi5WifiMode">
												<el-option label="" value=""></el-option>
												<el-option label="11a" value="AONLY"></el-option>
												<el-option label="11a/n" value="AN"></el-option>
												<el-option label="11a/an/ac" value="A_AN_AC"></el-option>
												<el-option label="11an/ac" value="AN_AC"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label="Channel" prop="wifi5WifiChannel">
											<el-select v-model="form.wifi5WifiChannel" size="mini">
												<el-option label="" value=""></el-option>
												<el-option label="AUTO" value="AUTO"></el-option>
												<el-option v-for="item in wifi5Chanel[form.wifi5WifiSupportChannel]" :label="item" :value="item"></el-option>
											</el-select>
										</el-form-item>
			 						</div>
			 						<div class="formContentWarp">
			 							<el-form-item label="Channel Bandwidth" prop="wifi5WifiBandwidth">
											<el-radio-group v-model="form.wifi5WifiBandwidth" style="margin-top: 6px;">
												<el-radio label="0">20M</el-radio>
												<el-radio label="1">40M</el-radio>
												<el-radio label="2">80M</el-radio>
												<el-radio label="3">160M</el-radio>
											</el-radio-group>
										</el-form-item>
										<el-form-item label="Support Channel" prop="wifi5WifiSupportChannel" style="margin-left: 196px;" >
											<el-select v-model="form.wifi5WifiSupportChannel" size="mini" @change="function(val){form.wifi5WifiChannel = '';}">
												<el-option label="" value=""></el-option>
												<el-option value="SC1" label="Ch36~48"></el-option>
												<el-option value="SC2" label="Ch36~64"></el-option>
												<el-option value="SC3" label="Ch52~64"></el-option>
												<el-option value="SC4" label="Ch149~161"></el-option>
												<el-option value="SC5" label="Ch149~165"></el-option>
												<el-option value="SC6" label="Ch36~48,Ch149~161"></el-option>
												<el-option value="SC7" label="Ch36~48,Ch149~165"></el-option>
												<el-option value="SC8" label="Ch36~64,Ch100~140"></el-option>
												<el-option value="SC9" label="Ch36~64,Ch149~161"></el-option>
												<el-option value="SC10" label="Ch52~64,Ch149~161"></el-option>
												<el-option value="SC11" label="Ch52~64,Ch149~165"></el-option>
												<el-option value="SC12" label="Ch36~64,Ch100~120,Ch149~161"></el-option>
												<el-option value="SC13" label="Ch36~64,Ch100~116,Ch132~140"></el-option>
												<el-option value="SC14" label="Ch36~64,Ch100~124,Ch149~161"></el-option>
												<el-option value="SC15" label="Ch36~64,Ch100~140,Ch149~161"></el-option>
												<el-option value="SC16" label="Ch36~64,Ch100~140,Ch149~165"></el-option>
												<el-option value="SC17" label="Ch52~64,Ch100~140,Ch149~161"></el-option>
												<el-option value="SC18" label="Ch56~64,Ch100~140,Ch149~161"></el-option>
												<el-option value="SC19" label="Ch36~64,Ch100~116,Ch132~140,Ch149~165"></el-option>
												<el-option value="SC20" label="Ch36~64,Ch100~116,Ch136~140,Ch149~165"></el-option>
											</el-select>
										</el-form-item>
			 						</div>
			 						
			 						<div style="margin-left: 14px; margin-bottom: 22px;">		 					
				 						<div style="margin-bottom: 10px; font-size: 14px; ">MBSSID</div>
										<el-ctable height="177" ref="taskList" :data="tbDataWifi5" :pagination="false" style="border: 1px solid #E9E9E9;width:80%; ">
											<el-table-column width="80">
												<template slot-scope="scope">
													<i class="el-icon el-icon-operation-edit" @click="modifyWifi5(scope.row)"  v-show="!isWifi5Closed && editBtnShow"></i>
												</template>
											</el-table-column>
											<el-table-column label="Network Name(SSID)" prop="wifi5WifiSsid"></el-table-column>
											<el-table-column label="Security Mode" prop="wifi5WifiEncryption">
												<template slot-scope="scope">
													{{modeKeys[scope.row.wifi5WifiEncryption]}}
												</template>
											</el-table-column>
											<el-table-column label="Status" prop="wifi5WifiEnable">
												<template slot-scope="scope">
													<span v-if="scope.row.wifi5WifiEnable=='1'">Enable</span>
													<span v-if="scope.row.wifi5WifiEnable=='0'">Disable</span>
												</template>
											</el-table-column>									
										</el-ctable>
				 					</div>
			 					</div>
			 					
			 					<!--DMZ  -->	
			 					<div>
				 					<div class="titleWarp">
				 						<div class="cpeCircleTip"></div>
				 						<span>DMZ</span>
				 					</div>				 							 					
				 					<div class="formContentWarp">
										<el-form-item label="DMZ Enable" prop="dmzEnable">
											<el-select v-model="form.dmzEnable" :disabled="readonly" class="selectOptions" > 
												<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label="DMZ Address" prop="dmzHostAddress" >
											<el-input v-model="form.dmzHostAddress" maxlength="100" :readonly="readonly"></el-input>
										</el-form-item>
				 					</div>
			 					</div>	
			 					<!--LAN  -->	
			 					<div>
				 					<div class="titleWarp">
				 						<div class="cpeCircleTip"></div>
				 						<span>LAN</span>
				 					</div>			 							 					
				 					<div style='margin-left: 14px;' class="lanWarp">
										<el-form-item label="LAN <%=rb.getString("JieKou")%>" prop="lanEnable">
											<el-select v-model="form.lanEnable" :disabled="readonly" class="selectOptions" > 
												<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
											</el-select>
										</el-form-item>
				 					</div>
			 					</div> 	
							</div>
							
							<!-- LTE -->
							<div v-show="activeCode=='lte'" style="min-height: 500px;">
								<!-- PCI Lock  -->
			 					<div v-show="cpeModel == '4G'">
				 					<div class="titleWarp">
				 						<div class="cpeCircleTip"></div>
				 						<span><%=rb.getString("SuoPCI")%></span>
				 					</div>				 							 					
				 					<div style="margin-left: 14px;" class="lteWidthWarp">
				 						<el-form-item label="<%=rb.getString("SaoMiaoFangShi")%>" prop="scanMode">
											<el-select v-model="form.scanMode" :disabled="readonly" class="selectScanMode"> 
												<el-option label="" value=""></el-option>
												<el-option label="Full Band" value="fullband"></el-option>
												<el-option label="Band/Frequency Preferred" value="freqpreferred"></el-option>
												<el-option label="PCI lock" value="pcilock"></el-option>
												<el-option label="PCI Only Lock" value="pcionlylock"></el-option>
											</el-select>
										</el-form-item>	
										
										<!-- earfcn -->
										<div v-if="form.scanMode == 'freqpreferred'">
											<el-form-item class="earfcn-pci-line" prop="" label='Earfcn' style="margin-bottom: 3px;">
												<div class="mainWarp">						
													<div class='newAddBtn' >
														<el-input v-model='form.onlyEarfcn' :disabled="readonly"></el-input>
														<i v-show="AddBtnShow" class="el-icon el-icon-plus" @click='addEarfcnBtn' style="margin-left: 10px;"></i>
													</div>							
												</div>
											</el-form-item>
											<div class="earfcnPciWarp">
												<el-form-item class='suffixItem' v-for='(domain,index) in form.earfcnList' style='line-height:16px;'>
													<div class='form-suffix'>
														<span class='text'>Earfcn : {{domain}}</span>
														<span v-show="AddBtnShow" style='font-size:16px;margin-top:2px;' class='form-bt-remove el-icon el-icon-operation-delete' @click.prevent='removeEarfcn(domain)'></span>
													</div>
												</el-form-item> 
												<p class="cpeAddErrorTip">{{earfcnErrorMessage}}</p> 
											</div>											
										</div>
										<!-- earfcn and pci -->
										<div v-if="form.scanMode == 'pcilock'">
											<el-form-item class="earfcn-pci-line" label='Earfcn : PCI' style="margin-bottom: 3px;">
												<div class="mainWarp">						
													<div class='newAddBtn'>
														<el-input v-model='form.earfcnStart' style='width:200px;' :disabled="readonly"></el-input> — <el-input v-model='form.pciEnd' style='width:200px;' :disabled="readonly"></el-input>		 																
														<i v-show="AddBtnShow" class="el-icon el-icon-plus addIpBtn" @click='addPcilockBtn' style="margin-left: 10px;"></i>
													</div>							
												</div>
											</el-form-item>
											<div class="earfcnPciWarp">
												<el-form-item class='suffixItem' v-for='(domain,index) in form.earfcnPciList' style='line-height:16px;'>
													<div class='form-suffix'>
														<div style="display: flex;">
															<span class='text'>Earfcn : {{domain.startValue}}</span>
															<span class='text' style="margin: 0 10px 0 15px;">PCI : {{domain.endValue}}</span>
															<span v-show="AddBtnShow" style='font-size:16px;margin-top:6px;' class='form-bt-remove el-icon el-icon-operation-delete' @click.prevent='removeEarfcnPci(domain)'></span>															
														</div>														 
													</div>
												</el-form-item> 
												<p class="cpeAddErrorTip">{{earfcnPciErrorMessage}}</p>
											</div>											
										</div>
										<!--  pci -->
										<div v-if="form.scanMode == 'pcionlylock'">
											<el-form-item class="earfcn-pci-line" label='PCI' style="margin-bottom: 3px;">
												<div class="mainWarp">						
													<div class='newAddBtn'>
														<el-input v-model='form.onlyPci' :disabled="readonly"></el-input>
														<i v-show="AddBtnShow" class="el-icon el-icon-plus" @click='addPciBtn' style="margin-left: 10px;"></i>
													</div>							
												</div>
											</el-form-item>
											<div class="earfcnPciWarp">
												<el-form-item class='suffixItem' v-for='(domain,index) in form.pciList' style='line-height:16px;'>
													<div class='form-suffix'>
														<span class='text'>PCI : {{domain}}</span>
														<span v-show="AddBtnShow" style='font-size:16px;margin-top:2px;' class='form-bt-remove el-icon el-icon-operation-delete' @click.prevent='removePci(domain)'></span>
													</div>
												</el-form-item> 
												<p class="cpeAddErrorTip">{{pciErrorMessage}}</p>
											</div>
										</div>																	
				 					</div>
									 <el-form-item v-show="false" prop="pci">
										<el-input v-model="form.pci"></el-input>
									</el-form-item>		 									 					
			 					</div>
								<div v-show="cpeModel == '5G'" >
									<!-- 5G -->
									<div class="titleWarp">
										<div class="cpeCircleTip"></div>
										<span><%=rb.getString("SuoPCI")%></span>
									</div>	
									<div style='position:relative;border:none;width:95%;margin-top:20px;margin-left:20px;'>
										<el-form-item label="<%=rb.getString("SaoMiaoFangShi")%>" prop="scanMode" class="selectScanMode">
											<el-select v-model="form.scanMode" :disabled='readonly' @change="scanModeChange">
												<el-option label="Full Band" value="fullband"></el-option>
												<el-option label="Frequency Lock" value="freqlock"></el-option>
												<el-option label="Cell Lock" value="celllock"></el-option>
												<el-option label="Band Lock" value="bandlock"></el-option>
											</el-select>
										</el-form-item>
										<!-- 5G Frequency lock -->
										<div v-show="form.scanMode=='freqlock'">
											<el-form-item label="Frequency Lock" style="margin-bottom: 5px;">
												<i v-if="!readonly" class="el-icon el-icon-plus" style="margin-top: 5px;" @click="showFreqLock"></i>
											</el-form-item>
											<el-table height="200px" :data="freqLockList">
												<el-table-column label="Index" type="index" width="100"></el-table-column>
												<el-table-column label="Rat" prop="rat">
													<template slot-scope="scope">
														<span v-if="scope.row.rat == '0'">4G LTE</span>
														<span v-if="scope.row.rat == '1'">5G NR</span>
													</template>
												</el-table-column>
												<el-table-column label="Band" prop="band"></el-table-column>
												<el-table-column label="Freq" prop="freq"></el-table-column>
												<el-table-column v-if="!readonly" label="Operation" width="100">
													<template slot-scope="scope">
														<i @click="deleteFreqLock(scope.$index)" class="el-icon el-icon-operation-delete"></i>
													</template>
												</el-table-column>
											</el-table>
										</div>
										<!-- 5G cell lock -->
										<div v-show="form.scanMode == 'celllock'">
											<el-form-item label="Cell Lock" style="margin-bottom: 5px;">
												<i v-if="!readonly" class="el-icon el-icon-plus" style="margin-top: 5px;" @click="showCellLock"></i>
											</el-form-item>
											<el-table height="200px" :data="cellLockList">
												<el-table-column label="Index" type="index" width="100"></el-table-column>
												<el-table-column label="Rat" prop="rat">
													<template slot-scope="scope">
														<span v-if="scope.row.rat == '0'">LTE</span>
														<span v-if="scope.row.rat == '1'">NR</span>
													</template>
												</el-table-column>
												<el-table-column label="Band" prop="band"></el-table-column>
												<el-table-column label="Earfcn" prop="earfcn"></el-table-column>
												<el-table-column label="PCI" prop="pci"></el-table-column>
												<el-table-column v-if="!readonly" label="Operation" width="100">
													<template slot-scope="scope">
														<i @click="deleteCellLock(scope.$index)" class="el-icon el-icon-operation-delete"></i>
													</template>
												</el-table-column>
											</el-table>
										</div>
						
										<!-- 5G Band lock -->
										<div v-show="form.scanMode == 'bandlock'">
											<el-form-item label="5G Band Lock" style="margin-bottom: 5px;">
												<i v-if="!readonly" class="el-icon el-icon-plus" style="margin-top: 5px;" @click="showBandLock"></i>
											</el-form-item>
											<el-table height="200px" :data="bandLockList">
												<el-table-column label="Index" type="index" width="100"></el-table-column>
												<el-table-column label="Rat" prop="rat">
													<template slot-scope="scope">
														<span v-if="scope.row.rat == '0'">LTE</span>
														<span v-if="scope.row.rat == '1'">SA</span>
														<span v-if="scope.row.rat == '2'">NSA</span>
													</template>
												</el-table-column>
												<el-table-column label="Band" prop="band"></el-table-column>
												<el-table-column v-if="!readonly" label="Operation" width="100">
													<template slot-scope="scope">
														<i @click="deleteBandLock(scope.$index)" class="el-icon el-icon-operation-delete"></i>
													</template>
												</el-table-column>
											</el-table>
										</div>
									</div>
								</div>
								<!-- apn 提示 -->
								<div style='padding: 30px 0 0 14px;' class='commonNotes12'>
									<i class="el-icon el-icon-circle-info" style='margin-right: 6px;'></i><%=rb.getString("APNPeiZhiTiShi")%>
								</div>
							</div>
							<!-- System -->
							<div v-show="activeCode=='system'">
								<!-- Web Access  -->
			 					<div >
				 					<div class="titleWarp">
				 						<div class="cpeCircleTip"></div>
				 						<span>Web Access</span>
				 					</div>				 							 					
				 					<div class="systemWarp">
				 						<el-form-item label="Password" prop="uiPassword">
											<el-input v-model="form.uiPassword" maxlength="100" :readonly="readonly"></el-input>
										</el-form-item>	
										<el-form-item label="Https<%=rb.getString("SheZhiKaiGuan")%>" prop="httpsEnable">
											<el-select v-model="form.httpsEnable" :disabled="readonly" class="selectOptions" > 
												<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label="HttpsWan<%=rb.getString("SheZhiKaiGuan")%>" prop="httpsWanEnable">
											<el-select v-model="form.httpsWanEnable" :disabled="readonly" class="selectOptions" > 
												<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label="Access Control <%=rb.getString("SheZhiKaiGuan")%>" prop="wanAccEnable">
											<el-select v-model="form.wanAccEnable" :disabled="readonly" class="selectOptions" > 
												<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item v-show="false" prop="rangeIp">
											<el-input v-model="form.rangeIp"></el-input>
										</el-form-item>	
										<div class="iplistTitle" style="width:81%; margin-top: -10px;">
											<span style="font-size: 14px; color: #333333;"><%=rb.getString("FangWenKongZhiLieBiao")%></span>
											<div class="circleIcon" style="top:0px;" v-show="AddBtnShow">
												<span class="el-icon el-icon-circle-add" @click="systemAddIp" ></span>
												<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
											</div>
										</div>	
										<div style="border:1px solid #E9E9E9;margin-bottom:30px;margin-left: 14px; width:80%;" class="ipTable">
											<el-table ref="ctableIp" :data="ipTableData.slice((currentPage-1)*pageSize,currentPage*pageSize)" height="200" border>
												<el-table-column label="" type="index" min-width="50"></el-table-column>
												<el-table-column label='' width="70" >
													<template slot-scope="scope">
														<div v-show="AddBtnShow" class="el-icon el-icon-operation-edit" title="<%=rb.getString("GengXin") %>" @click="updateIp(scope.row,scope.$index)"></div>
														<div v-show="AddBtnShow" class="el-icon el-icon-operation-delete" title="<%=rb.getString("ShanChu") %>" @click="deleteIp(scope.row,scope.$index)"></div>									
													</template>
												</el-table-column>												
												<el-table-column label="<%=rb.getString("KaiShiIP") %>" prop="ipStart"></el-table-column>
												<el-table-column label="<%=rb.getString("JieShuIP") %>" prop="ipEnd"></el-table-column>
											</el-table>
											<el-pagination 
												@size-change="handleSizeChange"
												@current-change="handleCurrentChange"
												:current-page="currentPage"
												:page-sizes="[50,100,200]"
												:pageSize="pageSize"
												layout="total,sizes, prev, pager, next, jumper"
												:total="ipTableData.length">
											</el-pagination>
										</div>							
				 					</div>					 								 									 					
			 					</div>
			 					<!-- Ping Watchdog  -->
			 					<div>
				 					<div class="titleWarp">
				 						<div class="cpeCircleTip"></div>
				 						<span>Ping Watchdog</span>
				 					</div>				 							 					
				 					<div class="systemWarp">
				 						<el-form-item label="Ping Watchdog <%=rb.getString("KaiGuan")%>" prop="watchDogEnable">
											<el-select v-model="form.watchDogEnable" :disabled="readonly" class="selectOptions" > 
												<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label="IP Address or URL to Ping" prop="watchDogPingIp" >
											<el-input v-model="form.watchDogPingIp" maxlength="45" placeholder="Length: 0-45" :readonly="readonly"></el-input>
										</el-form-item>
										<el-form-item label="Ping Timeout(Seconds)" prop="watchDogPingTimeout">
											<el-input v-model="form.watchDogPingTimeout" maxlength="8" placeholder="Range: 1-65535" :readonly="readonly"></el-input>
										</el-form-item>
										<el-form-item label="Ping Count" prop="watchDogPingCount">
											<el-input v-model="form.watchDogPingCount" maxlength="8" placeholder="Range: 1-65535" :readonly="readonly"></el-input>
										</el-form-item>
										<el-form-item label="Failure Count to Reboot" prop="watchDogFailureReboot">
											<el-input v-model="form.watchDogFailureReboot" maxlength="8" placeholder="Range: 1-65535" :readonly="readonly"></el-input>
										</el-form-item>									
				 					</div>				 									 					
			 					</div>
			 					<!--snmp setting  -->
			 					<div>
				 					<div class="titleWarp">
				 						<div class="cpeCircleTip"></div>
				 						<span>SNMP</span>
				 					</div>				 							 					
				 					<div class="systemWarp">
										<el-form-item label="SNMP <%=rb.getString("KaiGuan")%>" prop="snmpEnable">
											<el-select v-model="form.snmpEnable" :disabled="readonly" class="selectOptions" > 
												<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label="NMS Address" prop="nmsAddress" >
											<el-input v-model="form.nmsAddress" maxlength="" placeholder="" :readonly="readonly"></el-input>
										</el-form-item>
										<el-form-item label="NMS Port" prop="nmsPort">
											<el-input v-model="form.nmsPort" maxlength="5" placeholder="" :readonly="readonly"></el-input>
										</el-form-item>
										<el-form-item label="Listening Port" prop="listeningPort">
											<el-input v-model="form.listeningPort" maxlength="5" placeholder="" :readonly="readonly"></el-input>
										</el-form-item>
										<el-form-item label="Trap Community" prop="trapCommunity">
											<el-input v-model="form.trapCommunity" maxlength="" placeholder="" :readonly="readonly"></el-input>
										</el-form-item>	
										<el-form-item label="Version" prop="version">
											<el-select v-model="form.version" :disabled="readonly" class="selectOptions" > 
									            <el-option label="" value=""></el-option>
									            <el-option label="V1&V2c" value="v2c"></el-option>
									    		<el-option label="V3" value="v3"></el-option>
									    	</el-select>
										</el-form-item>		
										<div v-show='form.version == "v2c"'>
											<el-form-item label="Read Community" prop="readCommunity">
												<el-input v-model="form.readCommunity" maxlength="" placeholder="" :readonly="readonly"></el-input>
											</el-form-item>
											<el-form-item label="RW Community" prop="rwCommunity">
												<el-input v-model="form.rwCommunity" maxlength="" placeholder=""></el-input>
											</el-form-item>
										</div>
										<div v-show='form.version == "v3"'>
											<el-form-item label="User Name" prop="userName">
												<el-input v-model="form.userName" maxlength="" placeholder="" :readonly="readonly"></el-input>
											</el-form-item>
											<el-form-item label="Authentication Protocol" prop="authenticationProtocol">
									    		<el-select v-model="form.authenticationProtocol" :disabled="readonly" class="selectOptions" > 
									            	<el-option label="" value=""></el-option>
									            	<el-option label="MD5" value="MD5"></el-option>
									    			<el-option label="SHA" value="SHA"></el-option>
									    		</el-select>
									   	 	</el-form-item>
											<el-form-item label="Authentication Passphrase" prop="authenticationPassphrase">
												<el-input v-model="form.authenticationPassphrase" maxlength="" placeholder="" :readonly="readonly"></el-input>
									    	</el-form-item>
											<el-form-item label="Privacy Protocol" prop="privacyProtocol">
												<el-select v-model="form.privacyProtocol" :disabled="readonly" class="selectOptions" > 
									           		<el-option label="" value=""></el-option>
									           		<el-option label="DES" value="DES"></el-option>
									    			<el-option label="AES" value="AES"></el-option>
									    		</el-select>
											</el-form-item>
											<el-form-item label="Privacy Passphrase" prop="privacyPassphrase">
												<el-input v-model="form.privacyPassphrase" maxlength="" placeholder="" :readonly="readonly"></el-input>
											</el-form-item>
										</div>							
				 					</div>				 									 					
			 					</div>
							</div>		 									
						</div>
					</div>
				</div>             
            </div>
        </div>  		
	</el-form>
	
	<div class='footer'>	
		<div>
			<el-button :disabled="readonly" type="primary" @click="formSubmit" :loading="submitLoading"><%=rb.getString("QueDing")%></el-button>
			<el-button :disabled="readonly" @click="cancel"><%=rb.getString("QuXiao")%></el-button>
		</div>		
	</div>
	
	<!-- 导入 弹窗 -->
	<el-dialog title="<%=rb.getString("DaoRu")%>" width="750px" :visible="showImportCard" class="cpeImportCard" :close-on-click-modal="false" :modal-append-to-body="false" @close="closeImportParams">		
		<el-form label-position="left" ref="importRuleForm" :model='importRuleForm' :rules='importRules'>     		     			            
        	<el-form-item label="<%=rb.getString("DaoRuWenJian")%>" label-width="110px" prop="fileName">
               	<el-upload 
             		ref="upload"
             		:before-upload='beforeUpload' 
             		:on-success='checkFile' 
             		:on-change="fileChange"  
             		:show-file-list=false 	                  		
				    :action="importRuleForm.uploadFileUrl" 
				    :data="fileParams" 
				    name="uploadFile" 
				    accept=".xls,.xlsx"
				    :auto-upload="false">
					<el-input :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' class="w400">
						<a slot="suffix" class="el-icon el-icon-operation-import importBox" @click="fileSelect"></a>
					</el-input>								
					<a slot="trigger" ref="file_up"></a>
					<span class="fileAcceptTip"><%=rb.getString("DangQianZhiChiWenJianLeiXing")%></span>
				</el-upload>	
         	</el-form-item> 
         	<el-form-item label="">
         		<div style='color:#999999;'>
         			<span class='el-icon el-icon-circle-info' style='font-size:14px;margin-right:5px;'></span>
         			<%=rb.getString("DaoRuWenJianTiShi")%>
					<span style="cursor:pointer;margin-left:10px;" @click="exportTemplate">
						<span style='' class='el-icon el-icon-common-download'></span>
						<span style='color:#363B4E;text-decoration:underline'><%=rb.getString("DaoChuMuBan")%></span>
					</span>
				</div> 
         	</el-form-item>      	    
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="uploadParams"><%=rb.getString("QueDing")%></el-button>
              <el-button @click="closeImportParams"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	
	<!-- MBSSID修改窗口 -->
	<el-dialog title="Modify MBSSID" :visible.sync="wifiMDLShow" :modal="true" :append-to-body="true" width="600">
		<el-form ref="modifyForm" :model="mForm" :rules="mFormRules" class="cpe-label-flex" label-width="160" style="padding-left: 20px;">
			<el-form-item v-show="false" prop="id">
				<el-input v-model="mForm.id"></el-input>
			</el-form-item>
			<el-form-item v-show="mForm.id != 'main'" label="Muti-SSID Status" prop="wifiEnable">
				<el-select v-model="mForm.wifiEnable">
					<el-option label="Enable" value="1"></el-option>
					<el-option label="Disable" value="0"></el-option>
				</el-select>
			</el-form-item>
			<div v-show="mForm.wifiEnable == '1'">
				<el-form-item label="Network Name(SSID)" prop="wifiSsid">
					<el-input v-model="mForm.wifiSsid" maxlength="100"></el-input>
				</el-form-item>
				<el-form-item label="Security Mode" prop="wifiEncryption">
					<el-select v-model="mForm.wifiEncryption">
						<el-option label="OPEN" value="OPEN"></el-option>
						<el-option label="WPAPSK" value="WPA"></el-option>
						<el-option label="WPA2PSK" value="WPA2"></el-option>
						<el-option label="WPAPSK/WPA2PSK" value="WPAWPA2"></el-option>
					</el-select>
				</el-form-item>
				<div v-show="mForm.wifiEncryption != 'open'">
					<el-form-item v-show="mForm.wifiEncryption && mForm.wifiEncryption!='open' && false" label="WPA Algorithm">
						{{wpakeys[mForm.wifiEncryption]}}
					</el-form-item>
					<div style="padding-bottom: 25px;">
						<el-checkbox v-model="mForm.showPassword">Display Password</el-checkbox>
						<el-form-item v-show="false" prop="showPassword"></el-form-item>
					</div>
					<el-form-item label="Pass Phrase" prop="wifiPassphrase">
						<el-input :type="mForm.showPassword==true?'text':'password'" v-model="mForm.wifiPassphrase"></el-input>
					</el-form-item>
				</div>
			</div>
		</el-form>
		<div slot="footer">
			<el-button type="primary" @click="saveModify"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="wifiMDLShow = false"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
	
	<!-- wifi5 MBSSID修改窗口 -->
	<el-dialog title="Modify MBSSID" :visible.sync="wifi5WifiMDLShow" :modal="true" :append-to-body="true" width="600">
		<el-form ref="modifyWifi5Form" :model="mWifi5Form" :rules="mWifi5FormRules" class="cpe-label-flex" label-width="160" style="padding-left: 20px;">
			<el-form-item v-show="false" prop="id">
				<el-input v-model="mWifi5Form.id"></el-input>
			</el-form-item>
			<el-form-item v-show="mWifi5Form.id != 'main'" label="Muti-SSID Status" prop="wifi5WifiEnable">
				<el-select v-model="mWifi5Form.wifi5WifiEnable">
					<el-option label="Enable" value="1"></el-option>
					<el-option label="Disable" value="0"></el-option>
				</el-select>
			</el-form-item>
			<div v-show="mWifi5Form.wifi5WifiEnable == '1'">
				<el-form-item label="Network Name(SSID)" prop="wifi5WifiSsid">
					<el-input v-model="mWifi5Form.wifi5WifiSsid" maxlength="100"></el-input>
				</el-form-item>

				<el-form-item label="Security Mode" prop="wifi5WifiEncryption">
					<el-select v-model="mWifi5Form.wifi5WifiEncryption">
						<el-option label="OPEN" value="OPEN"></el-option>
						<el-option label="WPAPSK" value="WPA"></el-option>
						<el-option label="WPA2PSK" value="WPA2"></el-option>
						<el-option label="WPAPSK/WPA2PSK" value="WPAWPA2"></el-option>
					</el-select>
				</el-form-item>
				<div v-show="mWifi5Form.wifi5WifiEncryption != 'open'">
					<el-form-item v-show="mWifi5Form.wifi5WifiEncryption && mWifi5Form.wifi5WifiEncryption!='open' && false" label="WPA Algorithm">
						{{wpakeys[mWifi5Form.wifi5WifiEncryption]}}
					</el-form-item>
					<div style="padding-bottom: 25px;">
						<el-checkbox v-model="mWifi5Form.wifi5ShowPassword">Display Password</el-checkbox>
						<el-form-item v-show="false" prop="wifi5ShowPassword"></el-form-item>
					</div>
					<el-form-item label="Pass Phrase" prop="wifi5WifiPassphrase">
						<el-input :type="mWifi5Form.wifi5ShowPassword==true?'text':'password'" v-model="mWifi5Form.wifi5WifiPassphrase"></el-input>
					</el-form-item>
				</div>
			</div>
		</el-form>
		<div slot="footer">
			<el-button type="primary" @click="saveModifyWifi5"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="wifi5WifiMDLShow = false"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
	
    <!-- 新建ip弹窗 -->
	<el-dialog title="<%=rb.getString("TianJia")%>" id="addIpDialog" width='600px' :visible.sync='addIpShow' :append-to-body="true" :close-on-click-modal="false" @close='closeAddIp' class="cpeIpModalWarp">
		<el-form ref='addIpForm' :model='addIpForm' label-position="left">
			<el-form-item label="IP" label-width="50px"  style='margin-bottom:0px;position:relative'>
				<el-input v-model='addIpForm.ipStart' style='width:200px;'></el-input> — <el-input v-model='addIpForm.ipEnd' style='width:200px;'></el-input>		 		
				<span @click='addIpBtn' class='form-bt el-icon el-icon-plus' style='vertical-align:middle'></span>
			</el-form-item>
			<div class="cpeIpAddressWarp">
				<el-form-item class='suffixItem' v-for='(domain,index) in addIpForm.ipGroup' style='line-height:16px;'>
					<div class='form-suffix'>
						<span class='text'>{{domain}}</span>
						<span style='font-size:16px;margin-top:2px;' class='form-bt-remove el-icon el-icon-operation-delete' @click.prevent='removeIp(domain)'></span>
					</div>
				</el-form-item> 
				<p class="cpeAddErrorTip">{{errorMessage}}</p>
			</div>
			<div class="cpeIpWarpFotter">
				<el-button @click='saveAddIp' type="primary"><%=rb.getString("QueDing")%></el-button>
				<el-button @click='closeAddIp'><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-form>
	</el-dialog>
	
	<!-- 修改ip弹窗 -->
	<el-dialog title="<%=rb.getString("XiuGai")%>" id="editIpDialog" width='600px' :visible.sync='editIpShow' :append-to-body="true" :close-on-click-modal="false" @close='closeEditIp' class="cpeIpModalWarp">
		<el-form ref='editIpForm' :model='editIpForm' label-position="left">
			<el-form-item label="IP" label-width="50px"  style='margin-bottom:0px;position:relative'>
				<el-input v-model='editIpForm.ipStart' style='width:200px;'></el-input> — <el-input v-model='editIpForm.ipEnd' style='width:200px;'></el-input>		 		
			</el-form-item>
			<p class="editErrorTip">{{editErrorMsg}}</p>			
		</el-form>
		<div  class="cpeIpWarpFotter">
			<el-button @click='saveEditIp' type="primary"><%=rb.getString("QueDing")%></el-button>
			<el-button @click='closeEditIp'><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
	<!-- 新增 5G锁频 Frequency Lock弹窗 -->
	<el-dialog title="Add 5G Frequency Lock" id="nrLockAddDialog" :visible.sync="freq5gDlShow" :modal="true" :append-to-body="true" :close-on-click-modal="false">
		<el-form ref="freqForm" :model="freqForm" :rules="lockRules" label-width="80" class="cpe-label-flex half-item">
			<el-form-item label="Rat" prop="rat">
				<el-select v-model="freqForm.rat" @change="freqLockRatChange">
					<el-option label="4G LTE" value="0"></el-option>
					<el-option label="5G NR" value="1"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label="Band" prop="band">
				<el-select v-model="freqForm.band" @change="freqLockBandChange" filterable>
					<el-option v-for="item in bandList" :label="item?item:'Full'" :value="item"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label="Freq" prop="freq" required>
				<el-input v-model="freqForm.freq"></el-input>
			</el-form-item>
		</el-form>
		
		<div slot="footer">
			<el-button type="primary" @click="addFreqLock"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="freq5gDlShow = false"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
	<!-- 新增 5G锁频 Cell Lock弹窗 -->
	<el-dialog title="Add 5G Cell Lock" :visible.sync="cell5gDlShow" id="nrLockAddDialog" :modal="true" :append-to-body="true" :close-on-click-modal="false">
		<el-form ref="cellForm" :model="cellForm" :rules="lockRules" label-width="80" class="cpe-label-flex half-item">
			<el-form-item label="Rat" prop="rat">
				<el-select v-model="cellForm.rat">
					<el-option label="LTE" value="0"></el-option>
					<el-option label="NR" value="1"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label="Band" prop="band">
				<el-select v-model="cellForm.band" filterable>
					<el-option v-for="item in bandList" :label="item?item:'Full'" :value="item"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label="Earfcn" prop="earfcn" required>
				<el-input v-model="cellForm.earfcn"></el-input>
			</el-form-item>
			<el-form-item label="PCI" prop="pci" required>
				<el-input v-model="cellForm.pci"></el-input>
			</el-form-item>
		</el-form>
		
		<div slot="footer">
			<el-button type="primary" @click="addCellLock"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="cell5gDlShow = false"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
	<!-- 新增 5G锁频 Band Lock弹窗 -->	
	<el-dialog title="Add 5G Band Lock" :visible.sync="band5gDlShow" id="nrLockAddDialog" :modal="true" :append-to-body="true" :close-on-click-modal="false">
		<el-form ref="bandForm" :model="bandForm" :rules="lockRules" label-width="80" class="cpe-label-flex half-item">
			<el-form-item label="Rat" prop="rat">
				<el-select v-model="bandForm.rat">
					<el-option label="LTE" value="0"></el-option>
					<el-option label="SA" value="1"></el-option>
					<el-option label="NSA" value="2"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label="Band" prop="band">
				<el-select v-model="bandForm.band" filterable>
					<el-option v-for="item in bandList" :label="item?item:'Full'" :value="item"></el-option>
				</el-select>
			</el-form-item>
		</el-form>
		
		<div slot="footer">
			<el-button type="primary" @click="addBandLock"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="band5gDlShow = false"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
</div>

<script type="text/javascript">
	/**
	*  页面编辑和只读模式通过readonly控制
	*  校验规则也由readonly决定
	**/
	var cpeFactoryResetAddTaskVue = new Vue({
		el: '#cpeFactoryResetAddTask',
		data(){
			var vm = this, reg = /^(\d+,?)+$/,
				createChanel = function(chStr) {
					var vm = this,
						list = chStr.split(','),
						arr = [];
	
					list.map(function(seg){
						var range = seg.split('~');
	
						for(var i = range[0] - 0; i <= range[1] - 0; i += 4) {
							arr.push(i+'');
						}
					});
	
					return arr;
				},
				validateDevice = function(rule,value,callback) { // 校验设备
						if(vm.selectAll == '1'){
							callback();
						}else{
							if(value.length == 0) {
								callback('<%=rb.getString("QingXuanZeSheBei")%>');
							}else {
								callback();
							}
						}					
					},		
					
					/* watchdog相关校验 */
					watchdogValid = function(rule,value,cb){
						var currVal = value,
							enable = vm.form.watchDogEnable;
						
						if(currVal) {
							if(isNaN(currVal) || currVal-1<0 || currVal-65535>0){
								cb('Range: 1-65535');
							}else{
								cb();
							}
						}else {
							if(enable == 1) {
								cb('Range: 1-65535');
							}else {
								cb();
							}
						}
					},
					/* watchdog ip校验 */
					watchdogValidateIP = function(rule,value,cb) {
						var currVal = value,
							enable = vm.form.watchDogEnable;
						
						if(currVal && currVal.length>45){
							cb('<%=rb.getString("SheBeiMingChengGuiZe")%>');
						}else{
							if(!currVal && enable == 1) {
								cb('<%=rb.getString("SheBeiMingChengGuiZe")%>');
							}else {
								cb();
							}
						}
					},
					validWiFiEnable = function(rule,value,cb){
						if(value == '1' || value == '0') {
							cb();
						}else {
							cb('<%=rb.getString("QingXuanZe")%>');
						}
					},
					validSsid = function(rule,value,cb){
						if(vm.mForm.wifiEnable != '1') {
							cb();
						}else if(value) {
							cb();
						}else {
							cb('<%=rb.getString("ShuRuBiTianXiang")%>');
						}
					},
					validEncryption = function(rule,value,cb){
						if(vm.mForm.wifiEnable != '1') {
							cb();
						}else if(value) {
							cb();
						}else {
							cb('<%=rb.getString("QingXuanZe")%>');
						}
					},
					validPassphrase = function(rule,value,cb){
						if(vm.mForm.wifiEnable == '1' && vm.mForm.wifiEncryption != 'open') {
							if(value && value.length >= 8 && value.length <= 64) {
								cb();
							}else {
								cb('<%=rb.getString("ZiFuChang")%>: 8 - 64');
							}
						}else {
							cb();
						}
					},
					validateFileName = function(rule,value,callback) {
			        	var value = vm.fileName; 
						if( value === '' || value === null || value === undefined) {
							callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
						}else if(!fileFormatMatch(value,"xlsx,xls")){
                            callback(new Error("<%=rb.getString("DangQianZhiChiWenJianLeiXing")%>"))
                        }else {
							callback();
						} 
					},
					validSsidWifi5 = function(rule,value,cb){
						if(vm.mWifi5Form.wifi5WifiEnable != '1') {
							cb();
						}else if(value) {
							cb();
						}else {
							cb('<%=rb.getString("ShuRuBiTianXiang")%>');
						}
					},
					validEncryptionWifi5 = function(rule,value,cb){
						if(vm.mWifi5Form.wifi5WifiEnable != '1') {
							cb();
						}else if(value) {
							cb();
						}else {
							cb('<%=rb.getString("QingXuanZe")%>');
						}
					},
					validPassphraseWifi5 = function(rule,value,cb){
						if(vm.mWifi5Form.wifi5WifiEnable == '1' && vm.mWifi5Form.wifi5WifiEncryption != 'open') {
							if(value && value.length >= 8 && value.length <= 64) {
								cb();
							}else {
								cb('<%=rb.getString("ZiFuChang")%>: 8 - 64');
							}
						}else {
							cb();
						}
					},
					portValid = function(rule,value,cb){
						var reg = /^(\d+)$/;
						if(vm.form.snmpEnable == '1') {
							if(value && reg.test(value)) {
								cb();
							}else {
								cb('<%=rb.getString("ZhengXing")%>,<%=rb.getString("ZiFuChang")%>: 0-5');
							}
						}else {
							if(value === '' || value === null || value === undefined) {
								cb();
							}else {
								if(reg.test(value)){
									cb();
								}else{
									cb('<%=rb.getString("ZhengXing")%>,<%=rb.getString("ZiFuChang")%>: 0-5');
								}
							}
						}						
					},
					passphraseValid = function(rule,value,cb){
						if(vm.form.version == 'v3'){
							if(vm.form.snmpEnable == '1') {
								if(value && value.length >= 8 ) {
									cb();
								}else {
									cb('<%=rb.getString("snmpTiShi")%>');
								}
							}else {
								if(value === '' || value === null || value === undefined) {
									cb();
								}else {
									if(value.length >= 8){
										cb();
									}else{
										cb('<%=rb.getString("snmpTiShi")%>');
									}
								}
							}
						}else{
							cb();
						}
					},
					validateEarfcn = function(rule,value,cb) {
						if(value !== '') {
							if(value - 0 < 0 || value - 3279156 > 0 || isNaN(value)) {
								cb('<%=rb.getString("5GPinDianFanWei")%>');
							}else {
								cb();
							}
						}else {
							cb('Please Input Earfcn');
						}
					},
					
					validatePCI = function(rule,value,cb) {
						if(value !== '') {
							if(value - 0 < 0 || value - 1007 > 0 || isNaN(value)) {
								cb('<%=rb.getString("5GSpecificPCIFanWei")%>');
							}else {
								cb();
							}
						}else {
							cb('Please Input PCI');
						}
					},
					validateFreq = function(rule,value,callback) {
						var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/,
							bandVal = vm.freqForm.band,
							ratType = vm.freqForm.rat,
							rangeStr = ratType == '0' ? vm.bandCounterpartFreq4GList[bandVal] : vm.bandCounterpartFreq5GList[bandVal],
							minVal = parseInt(rangeStr.split('~')[0]),
							maxVal = parseInt(rangeStr.split('~')[1]);
		
						if(value == '' || value == undefined || value == null) {
							callback('<%=rb.getString("FanWei")%>：'+ rangeStr +',Integer');
						}else {
							if(reg.test(value) && value >= minVal && value <= maxVal) {
								callback();
							}else {
								callback('<%=rb.getString("FanWei")%>：'+ rangeStr +',Integer');
							}
						}
					};

			return {	        		       
				taskId:'',
				queryParams: {
					searchText: '',
					timeZone: timeZone,
					cpe_model: '4G',
				},
				height:'370px',				
				// 表单数据 
				wifi5Chanel: {
					''    : [],
					'SC1' : createChanel('36~48'),
					'SC2' : createChanel('36~64'),
					'SC3' : createChanel('52~64'),
					'SC4' : createChanel('149~161'),
					'SC5' : createChanel('149~165'),
					'SC6' : createChanel('36~48,149~161'),
					'SC7' : createChanel('36~48,149~165'),
					'SC8' : createChanel('36~64,100~140'),
					'SC9' : createChanel('36~64,149~161'),
					'SC10': createChanel('52~64,149~161'),
					'SC11': createChanel('52~64,149~165'),
					'SC12': createChanel('36~64,100~120,149~161'),
					'SC13': createChanel('36~64,100~116,132~140'),
					'SC14': createChanel('36~64,100~124,149~161'),
					'SC15': createChanel('36~64,100~140,149~161'),
					'SC16': createChanel('36~64,100~140,149~165'),
					'SC17': createChanel('52~64,100~140,149~161'),
					'SC18': createChanel('56~64,100~140,149~161'),
					'SC19': createChanel('36~64,100~116,132~140,149~165'),
					'SC20': createChanel('36~64,100~116,136~140,149~165')
				},
				form: { 
					//必传参数
					timeZone: timeZone,
					taskName: '${addTaskName}',
					cpeCodes: '',
					executeType: 'active',	
					time: '',
					taskId: '',					
					//network wifi
					wlanEnable: '',
					wifiChannel: '',
					wifiBandwidth: '', 					
					wifiSsid: '',
					wifiEncryption: '',
					wifiPassphrase: '',
					
					wifi1Enable: '',
					wifi1Ssid: '',
					wifi1Encryption: '',
					wifi1Passphrase: '',
					
					wifi2Enable: '',
					wifi2Ssid: '',
					wifi2Encryption: '',
					wifi2Passphrase: '',
					
					wifi3Enable: '',
					wifi3Ssid: '',
					wifi3Encryption: '',
					wifi3Passphrase: '',
					//wifi5
					wifi5WlanEnable: '', //wifi5 enable
					wifi5WifiMode: '', //network mode
					wifi5WifiChannel: '',//channel
					wifi5WifiBandwidth: '',//channel bandwith
					wifi5WifiSupportChannel: '',//support channel 
					
					wifi5WifiSsid: '',
					wifi5WifiEncryption: '',
					wifi5WifiPassphrase: '',

					wifi5Wifi1Enable: '',
					wifi5Wifi1Ssid: '',
					wifi5Wifi1Encryption: '',
					wifi5Wifi1Passphrase: '',
					
					wifi5Wifi2Enable: '',
					wifi5Wifi2Ssid: '',
					wifi5Wifi2Encryption: '',
					wifi5Wifi2Passphrase: '',
					
					wifi5Wifi3Enable: '',
					wifi5Wifi3Ssid: '',
					wifi5Wifi3Encryption: '',
					wifi5Wifi3Passphrase: '',
					//network dmz
					dmzEnable: '',
					dmzHostAddress: '',	
					//network lan
					lanEnable: '',
					//lte	pci lock				
					scanMode: '',
					pci: '',
					onlyEarfcn: '',
					earfcnStart: '',
					pciEnd: '',
					onlyPci: '',
					earfcnList: [],
					earfcnPciList: [],
					pciList: [],																	
					//system web access
					uiPassword: '',
					httpsEnable: '',
					httpsWanEnable: '',
					wanAccEnable: '',
					rangeIp: '',
					//system ping watchdog
					watchDogEnable: '',
					watchDogPingIp: '',
					watchDogPingTimeout: '',
					watchDogPingCount: '',
					watchDogFailureReboot: '',
					
					snmpEnable: '',
					nmsAddress: '',
					nmsPort: '',
					listeningPort: '',
					trapCommunity: '',
					version: '',
					readCommunity: '',
					rwCommunity: '',
					userName: '',
					authenticationProtocol: '',
					authenticationPassphrase: '',
					privacyProtocol: '',
					privacyPassphrase: '',

					macFilterEnable:'0',
					macFilterMode:'1',
					macAddMode:'0',
					macAddress:'',
					macFilterWhiteList:[],
					macFilterBlackList:[],

					ipFilterEnable:'0',
					ipFilterMode:'1',
					ipAddMode:'0',
					sourceIp:'',
					ipFilterWhiteList:[],
					ipFilterBlackList:[],

					urlFilterEnable:'0',
					urlFilterMode:'0',
					urlAddMode:'0',
					url:'',
					urlFilterBlackList:[],
				},
				cpeModel:'4G',
				macAddressErrorMessage:'',
				macAddressAddUploadFileURL:'',
				macAddressAddFileParams:{},
				macAddressAddFileName:'',

				sourceIpErrorMessage:'',
				sourceIpAddUploadFileURL:'',
				sourceIpAddFileParams:{},
				sourceIpAddFileName:'',

				urlErrorMessage:'',
				urlAddUploadFileURL:'',
				urlAddFileParams:{},
				urlAddFileName:'',

				earfcnErrorMessage: '',
				earfcnPciErrorMessage: '',
				pciErrorMessage: '',
				earfcnTip: false,
				editBtnShow: true,
				AddBtnShow: true,
				setDefault: '1', 
				selectAll:'0',
				// 校验规则
				rules: { 
					taskName:[
						{required: true,message:'<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'},
						{type:'string',max: 100,message:'<%=rb.getString("ZuiDaChangDu")%><%=rb.getString("MaoHao")%> 100'}
					],
					cpeCodes:[
						{validator: validateDevice}
					], 
					watchDogPingIp:[{validator: watchdogValidateIP}],
					watchDogPingTimeout:[{validator: watchdogValid}],
					watchDogPingCount:[{validator: watchdogValid}],
					watchDogFailureReboot:[{validator: watchdogValid}],
					nmsPort:[{validator: portValid}],
					listeningPort:[{validator: portValid}],
					authenticationPassphrase:[{validator: passphraseValid}],
					privacyPassphrase:[{validator: passphraseValid}]
				},
				showPairGrid: true,
		    	leftUrl : '',
		    	rightUrl : '',
				deviceTitle:['','<%=rb.getString("YiXuan")%>'],
				groupOptions:[],
				versionOptions:[],
				readonly: false,
				isPswModel: true,
				operateType:'',
				selection:[],							
				//参数配置
                activeCode: 'basic',
                selectOptions:[   				
    				{ text:'', value:''},
    				{ text:'Disable', value:'0'},
    				{ text:'Enable', value:'1'}
    			],
                
                profileListData: [],
                //ip table
                ipTableData: [],
                isLANEnable: true,
				//add ip
				addIpShow: false,
				addIpForm:{
					ipStart: '',
					ipEnd: '',
					ipGroup: []
				},
				errorMessage: '',
				//edit ip
				editIpShow: false,
				editIpForm:{
					ipStart: '',
					ipEnd: ''
				},
				editIndex: '',
				pageSize: 50,
				currentPage: 1,
				editErrorMsg: '',
				oldEditIpStart: '',
				oldEditIpEnd: '',
				notSupportShow: false,
				//end ip
				//import
				showImportCard:false,
				importRuleForm: {
		            uploadFileUrl: ''
		       	},	         	          
		        fileParams:{},              
		        fileName:'',	            					
				showFileTip:false,
				fileList:[],
				filePath:'',
				importRules: {	           		
					fileName:[{validator: validateFileName}]                 
		        },
		        //wifi
		        tbData: [],
				wifiMDLShow: false,
				mForm: {
					id: '',
					wifiEnable: '',
					wifiSsid: '',
					wifiEncryption: '',
					wifiPassphrase: '',
					showPassword: false
				},
				mFormRules: {
					wifiEnable: [{validator: validWiFiEnable}],
					wifiSsid: [{validator: validSsid}],
					wifiEncryption: [{validator: validEncryption}],
					wifiPassphrase: [{validator: validPassphrase}]
				},
				//wifi5
				tbDataWifi5: [],
				wifi5WifiMDLShow: false,
				mWifi5Form: {
					id: '',
					wifi5WifiEnable: '',
					wifi5WifiSsid: '',
					wifi5WifiEncryption: '',
					wifi5WifiPassphrase: '',
					wifi5ShowPassword: false
				},
				mWifi5FormRules: {
					wifi5WifiEnable: [{validator: validWiFiEnable}],
					wifi5WifiSsid: [{validator: validSsidWifi5}],
					wifi5WifiEncryption: [{validator: validEncryptionWifi5}],
					wifi5WifiPassphrase: [{validator: validPassphraseWifi5}]
				},
				wpakeys: {
					'OPEN': '',
					'WPA': 'TKIP',
					'WPA2': 'AES',
					'WPAWPA2': 'TKIP/AES'
				},
				modeKeys: {
					'OPEN': 'OPEN',
					'WPA': 'WPAPSK',
					'WPA2': 'WPA2PSK',
					'WPAWPA2': 'WPAPSK/WPA2PSK'
				},
				activeName:'',
				activeCollapse:['Mac','IP','URL'],
				freqLockList: [],
				cellLockList: [],
				bandLockList: [],
				freq5gDlShow: false,
				cell5gDlShow: false,
				band5gDlShow: false,
				freqForm: {
					rat: '0',
					band: '1',
					freq: '',
				},
				cellForm: {
					rat: '',
					band: '',
					earfcn: '',
					pci: ''
				},
				bandForm: {
					rat: '',
					band: ''
				},
				lockRules: {
					rat: [{required: true, message: 'Please Select Rat'}],
					band: [{required: true, message: 'Please Select Band'}],
					earfcn: [{validator: validateEarfcn}],
					pci: [{validator: validatePCI}],
					freq: [{validator: validateFreq}]
				},
				bandCounterpartFreq4GList:{
					'1':'0~599','2':'600~1199','3':'1200~1949','4':'1950~2399','5':'2400~2649','7':'2750~3449','8':'3450~3799','12':'5010~5179',
					'13':'5180~5279','14':'5280~5379','17':'5730~5849','18':'5850~5999','19':'6000~6149','20':'6150~6449','25':'8040~8689',
					'26':'8690~9039','28':'9210~9659','29':'9660~9769','30':'9770~9869','32':'9920~10359','34':'36200~36349','38':'37750~38249',
					'39':'38250~38649','40':'38650~39649','41':'39650~41589','42':'41590~43589','43':'43590~45589','46':'46790~54539','48':'55240~56739',
					'66':'66436~67335','71':'68586~68935'
				},
				bandCounterpartFreq5GList:{
					'1':'422000~434000','2':'386000~398000','3':'361000~376000','5':'173800~178800','7':'524000~538000','8':'185000~192000',
					'12':'145800~149200','13':'149200~151200','14':'151600~153600','18':'172000~175000','20':'158200~164200','25':'386000~399000',
					'26':'171800~178800','28':'151600~160600','29':'65535~65535','30':'470000~472000','38':'514000~524000','40':'460000~480000',
					'41':'499200~537999','48':'636667~646666','66':'422000~440000','70':'399000~404000','71':'123400~130400','75':'286400~303400',
					'76':'285400~286400','77':'620000~680000','78':'620000~653333','79':'693334~733333'
				},
                submitLoading:false,
			}	
		},
		computed: {
			formRules() { // 只读模式置空校验
				return this.readonly? []:this.rules;
			},

			tabs() {
				return [
					{code: 'basic', text: '<%=rb.getString("JiBenSheZhi")%>',show: true},				
					{code: 'network', text: '<%=rb.getString("WangLuoSheZhi")%>',show: true},
					{code: 'lte', text: 'LTE/NR',show: true},
					{code: 'system', text: '<%=rb.getString("XiTong")%>',show: true},
				]
			},
			isWifiClosed() {
				return this.form.wlanEnable != '1';
			},
			isWifi5Closed() {
				return this.form.wifi5WlanEnable != '1';
			},
			bandList() {
				var vm = this,
                	scanMode = vm.form.scanMode;
				let list = [];
				if(scanMode == 'celllock' || scanMode == 'bandlock'){
					for(let i=0;i<=100;i++) {
						list.push(i);
					}
				}else if(scanMode == 'freqlock'){
					var ratType = vm.freqForm.rat;
					if(ratType == '0'){
						list = ['1','2','3','4','5','7','8','12','13','14','17','18','19','20','25','26','28','29','30','32','34','38','39','40','41','42','43','46','48','66','71'];
					}else if(ratType == '1'){
						list = ['1','2','3','5','7','8','12','13','14','18','20','25','26','28','29','30','38','40','41','48','66','70','71','75','76','77','78','79'];
					}
				}
				return list;
			}
		},
		watch: {
			'form.cpeCodes': {
				handler: function(val){
					this.$refs.addform.validateField('cpeCodes');
				},
				deep: true
			},
			freqLockList: {
				handler: function(rows) {
					var vm = this,
						list = [];

					(rows||[]).map(function(row){
						list.push(row.rat +','+ row.band +','+ row.freq);
					});

					vm.form.pci = list.join(';');
				},
				deep: true
			},
			cellLockList: {
				handler: function(rows) {
					var vm = this,
						list = [];

					(rows||[]).map(function(row){
						list.push(row.rat +','+ row.band +','+ row.earfcn +','+ row.pci);
					});

					vm.form.pci = list.join(';');
				},
				deep: true
			},
			bandLockList: {
				handler: function(rows) {
					var vm = this,
						list = [];

					(rows||[]).map(function(row){
						list.push(row.rat +','+ row.band);
					});

					vm.form.pci = list.join(';');
				},
				deep: true
			}
		},
		methods: {						
			//参数配置
        	tabClick(tab) {
				var vm = this;
				if(vm.activeCode != tab.code) {
					vm.activeCode = tab.code;				
				}
			},
			// 初始化任务信息
			init(row,operateType,activeName) {
				var vm = this; 				
				vm.activeName = activeName;
				vm.taskId = row.TASK_ID;
				vm.operateType = operateType;				
				vm.readonly = vm.operateType == 'information' ? true : false;
							
				if(vm.readonly == true){
					vm.AddBtnShow = false;
					vm.editBtnShow = false;
				}		
				if(vm.operateType == 'information'){
					vm.showPairGrid = false;
					vm.leftUrl = '${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect=3';
					vm.rightUrl="${ctx}/cpe/batchconfig/getSelectedCPEInfo.action?taskId="+vm.taskId;	
					//详情
					vm.initFormInfo();
				}else{
					//新建
					vm.cpeModel = cpeFactoryResetTaskVue.cpe_model;
					vm.queryParams.cpe_model = vm.cpeModel;
					vm.$nextTick(function(){
						vm.leftUrl = '${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect=3';
					})	
					if(cpeFactoryResetTaskVue.selection.length > 0){
						vm.$nextTick(function(){
							vm.$refs.deviceListPairgrid.appendCheckedRows(cpeFactoryResetTaskVue.selection);
						})
					}
					vm.tbData = [];
					vm.reloadTable();
					vm.tbDataWifi5 = [];
					vm.wifi5ReloadTable();
					// 初始化form原始值
					vm.$nextTick(function(){
						initForm(vm.$refs.addform);					
					}); 
				}			
			},
			
			//详情页面回显form
			initFormInfo(){
				var vm = this;										
				if(vm.operateType == 'information'){	
					var params = {taskId: vm.taskId, timeZone: timeZone};
					axios.post('${ctx}/cpe/batchconfig/getTaskInfo.action', stringify(params)).then(function(response){
						var data = response.data;	
						if(data){
							vm.form.taskName = data.TASK_NAME;
							vm.form.executeType = data.EXECUTE_TYPE;
							if(data.SELECT_ALL == '0'){
								vm.selectAll = '0';
							}else{
								vm.selectAll = '1';
							}	
							vm.tbData = [];
							vm.tbDataWifi5 = [];
							if(data.paramConfig  === '' || data.paramConfig  === null || data.paramConfig === undefined){
								vm.reloadTable();
								vm.wifi5ReloadTable();
							}else{
								var curParams = JSON.parse(data.paramConfig);
 
								for(var key in curParams){	
									if(key == 'cpeType'){
										if(curParams['cpeType'] == "5"){
											vm['cpeModel'] = '5G';
										}else{
											vm['cpeModel'] = '4G';
										}
									}
									if(key == 'wifi'){
										if(curParams[key] === null || curParams[key] === '' || curParams[key] === undefined){
											//wifi 返回值为null,并不是对象
											vm.reloadTable();
										}else{
											if(curParams[key] && curParams[key]['wlanEnable']){
												vm.raloadTableData( curParams[key]);
											}else{
												vm.reloadTable();
											}											
										}
									}
									if(key == 'wifi5'){
										vm.tbDataWifi5 = [];
										if(curParams[key] === null || curParams[key] === '' || curParams[key] === undefined){
											vm.wifi5ReloadTable();
										}else{
											if(curParams[key] && curParams[key]['wifi5WlanEnable']){
												vm.wifi5RaloadTableData( curParams[key]);
											}else{
												vm.wifi5ReloadTable();
											}
										}
									}
									if(['ipFilter','macFilter','urlFilter'].indexOf(key) > -1){
										if(curParams[key] === null || curParams[key] === '' || curParams[key] === undefined){
											//全部为空数组
										}else{
											if(key == 'ipFilter'){
												['ipFilterBlackList','ipFilterWhiteList'].map(function(item){
													var ipList = curParams[key][item] ? curParams[key][item].split(';') : [],
														ipDataList = [];
													ipList.map((item,index)=>{
														ipDataList.push({
															sourceIp:item,
															paramsType:'default'
														});
													});
													curParams[key][item] = ipDataList
												})
											}
											if(key == 'macFilter'){
												['macFilterBlackList','macFilterWhiteList'].map(function(item){
													var macList = curParams[key][item] ? curParams[key][item].split(';') : [],
														macDataList = [];
													macList.map((item,index)=>{
														macDataList.push({
															macAddress:item,
															paramsType:'default'
														});
													});
													curParams[key][item] = macDataList;
												})
											}
											if(key == 'urlFilter'){
												['urlFilterBlackList','urlFilterWhiteList'].map(function(item){
													var urlList = curParams[key][item] ? curParams[key][item].split(';') : [],
														urlDataList = [];
													urlList.map((item,index)=>{
														urlDataList.push({
															url:item,
															paramsType:'default'
														});
													});
													curParams[key][item] = urlDataList;
												})
											}
										}
									}
								} 
								vm.reloadInfoForm(curParams);
							}
						}
					}).catch(function(error){})	
				}
			},
			
			// 打开导入弹出框
			importFileBtn(){				
				this.showImportCard = true;
			},
			//导入
			checkFile(res,file){  
				var vm = this;
				if(res.success){
					if(res.suc_count>0){
						vm.$message({
							type: 'success',
							message: '<%=rb.getString("ChengGong")%>'
						});
					}else {
						vm.$message({
							type: 'warning',
							message: '<%=rb.getString("ShiBai")%>'
						});
					}				
					vm.closeFileSelect();
					vm.showImportCard = false;
				}else{
					vm.$message({
						type: 'error',
						message: res.msg
					});
				}
				//修改已选择文件状态  
				var fileList = vm.$refs.upload.uploadFiles;
				fileList.forEach(function(file){
					file.status = 'ready';
				})
			},
	    	
			/**
			* 选择文件后，校验格式，并赋值页面显示 
			* @param file{object}   文件信息
			* @param fileList{Array}  文件列表
			*/ 
			fileChange(file,fileList){ 
				var vm = this;
				vm.fileName = file.name;
				vm.fileParams.FileName = file.name;
			},
			
			// 选择文件
			fileSelect(){  
				var vm =this;
				vm.$refs.upload.clearFiles();
				vm.$refs['file_up'].click();
			},
			
			// 移除导入文件
			closeFileSelect(){
				var vm = this;
				vm.fileName = '';			
				vm.$refs.upload.clearFiles();
			},
						
			/**
			* 文件上传之前
			* @param file{object}   文件信息
			*/ 
			beforeUpload(file){
				var vm = this, importUrl, fileName = file.name,
					fd = new FormData(),
					config = {
						headers: { 'Content-Type': 'multipart/form-data' }
					};

				fd.append('uploadFile',file); //文件流
				axios.post('${ctx}/cpe/batchconfig/importBatchConfigurationInfos.action',fd,config).then(function(res){
					var data = res.data;
					
					if(data["success"]){	
						vm.$message.success('<%=rb.getString("ChengGong")%>');						
						var curParams = data.paramConfig;
						vm.tbData = [];
						vm.tbDataWifi5 = [];
						var resetBefore = {
								taskName: vm.form.taskName,
								cpeCodes: vm.form.cpeCodes,								
							}
						vm.$refs.addform.resetFields(); //表单置空重新渲染,所有字段都被重置（x）
							//其它模块数据重新赋值
						Object.assign(vm.form, resetBefore);
						vm.closeImportParams();						
						if(data.paramConfig === '' || data.paramConfig === null || data.paramConfig === undefined){
							vm.reloadTable();
							vm.wifi5ReloadTable();
						}else{
							for(var key in curParams){
								vm.ipTableData = [];
								if(key == 'wifi'){
									vm.tbData = [];
									if(curParams[key] === null || curParams[key] === '' || curParams[key] === undefined){
										//wifi 返回值为null,并不是对象
										vm.reloadTable();
									}else{
										if(curParams[key] && curParams[key]['wlanEnable']){
											vm.raloadTableData( curParams[key]);
										}else{
											vm.reloadTable();
										}
									}
								}
								if(key == 'wifi5'){
									vm.tbDataWifi5 = [];
									if(curParams[key] === null || curParams[key] === '' || curParams[key] === undefined){
										//wifi 返回值为null,并不是对象
										vm.wifi5ReloadTable();
									}else{
										//wifi5
										if(curParams[key] && curParams[key]['wifi5WlanEnable']){
											vm.wifi5RaloadTableData( curParams[key]);
										}else{
											vm.wifi5ReloadTable();
										}
									}
								}
								if(['ipFilter','macFilter','urlFilter'].indexOf(key) > -1){
									if(curParams[key] === null || curParams[key] === '' || curParams[key] === undefined){
										//全部为空数组
									}else{
										if(key == 'ipFilter'){
											['ipFilterBlackList','ipFilterWhiteList'].map(function(item){
												var ipList = curParams[key][item] ? curParams[key][item].split(';') : [],
													ipDataList = [];
												ipList.map((item,index)=>{
													ipDataList.push({
														sourceIp:item,
														paramsType:'default'
													});
												});
												curParams[key][item] = ipDataList
											})
										}
										if(key == 'macFilter'){
											['macFilterBlackList','macFilterWhiteList'].map(function(item){
												var macList = curParams[key][item] ? curParams[key][item].split(';') : [],
													macDataList = [];
												macList.map((item,index)=>{
													macDataList.push({
														macAddress:item,
														paramsType:'default'
													});
												});
												curParams[key][item] = macDataList;
											})
										}
										if(key == 'urlFilter'){
											['urlFilterBlackList','urlFilterWhiteList'].map(function(item){
												var urlList = curParams[key][item] ? curParams[key][item].split(';') : [],
													urlDataList = [];
												urlList.map((item,index)=>{
													urlDataList.push({
														url:item,
														paramsType:'default'
													});
												});
												curParams[key][item] = urlDataList;
											})
										}
									}
								}
							}
							vm.reloadInfoForm(curParams);
						}
					}else{
						vm.$message.error(data["msg"])
	        			vm.fileList = [];
						vm.fileName = '';
						vm.$refs.importRuleForm.resetFields();
					} 
				})			
				return false;
			},
			
			reloadInfoForm(curParams){
				var vm = this;
				if(curParams != null || curParams != undefined){
					for(var key in curParams){
						if(typeof curParams[key] === 'object'){
							if(['ipFilterBlackList','ipFilterWhiteList','macFilterBlackList','macFilterWhiteList','urlFilterBlackList'].indexOf(key) > -1){
								vm.form[key] = curParams[key];
							}else{
								vm.reloadInfoForm(curParams[key])		
							}				
						}else{
							vm.form[key] = curParams[key];
							
							if(key == 'pci'){
								if( curParams[key] === '' || curParams[key] === null || curParams[key] === undefined){
									//全部为空数组
								}else{
									var newEarfcnPci = curParams[key].split(';');
									if(vm.cpeModel == '4G'){
										if(vm.form.scanMode == 'freqpreferred'){	
											vm.form.earfcnList = newEarfcnPci.map(function(item){												
												return item
											});
										}else if(vm.form.scanMode == 'pcilock'){
											vm.form.earfcnPciList = newEarfcnPci.map(function(item){
												var getRowData = {};
												if(item.includes(',')){
													getRowData = {     						
														startValue : item.split(',')[0],
														endValue : item.split(',')[1]	    			
													  }
												}
												return getRowData
											});
										}else if (vm.form.scanMode == 'pcionlylock'){
											vm.form.pciList = newEarfcnPci.map(function(item){
												return item
											});
										}
									}else{
										if(vm.form.scanMode == 'celllock') {
											newEarfcnPci.map(function(item){
												let arr = item.split(',');

												vm.cellLockList.push({
													rat: arr[0],
													band: arr[1],
													earfcn: arr[2],
													pci: arr[3]
												});
											});
										}else if(vm.form.scanMode == 'bandlock') {
											newEarfcnPci.map(function(item){
												let arr = item.split(',');

												vm.bandLockList.push({
													rat: arr[0],
													band: arr[1]
												});
											});
										}else if(vm.form.scanMode == 'freqlock') {
											newEarfcnPci.map(function(item){
												let arr = item.split(',');

												vm.freqLockList.push({
													rat: arr[0],
													band: arr[1],
													freq: arr[2]
												});
											});
										}
									}
								}
							}
							if(key == 'rangeIp'){								
								if(curParams[key] === '' || curParams[key] === null || curParams[key] === undefined){
									vm.ipTableData = [];
								}else{
									var newIpListData = curParams[key].split(';');
									if(newIpListData != '' || newIpListData.length > 0){
										vm.ipTableData = newIpListData.map((item,index) => {
											if(item.includes('-')){
												return {
													ipStart : item.split('-')[0],
						      						ipEnd : item.split('-')[1]
						      		    		}
											}else{
												return {
													ipStart : item
							      		    	}
											}
										});	
									}
								}
							}	
						}
					}
				} 
			},
		
			reloadTable(){
				var vm = this;
				vm.tbData.push({
					id: 'main',
					wifiEnable: '',
					wifiSsid: '',
					wifiEncryption: '',
					wifiPassphrase: ''
				}); 
				// Form更新
				Object.assign(vm.form,{
					wifiSsid: '',
					wifiEncryption: '',
					wifiPassphrase: ''
				});
				// sub 
				 ['1','2','3'].map(function(item){
					var row = {},pre = 'wifi',
						enableKey = pre+item+'Enable',
						idKey = pre+item+'Ssid',
						encryKey = pre+item+'Encryption',
						phraKey = pre+item+'Passphrase';							
					
					row['id'] = item;
					row['wifiEnable'] = '';
					row['wifiSsid'] = '';
					row['wifiEncryption'] = '';
					row['wifiPassphrase'] = '';

					// Form更新
					vm.form[enableKey] = '';
					vm.form[idKey] = '';
					vm.form[encryKey] = '';
					vm.form[phraKey] = '';
					vm.tbData.push(row); 
				}); 
			},
			
			raloadTableData( data){
				var vm = this;
				vm.tbData.push({
					id: 'main',
					wifiEnable: data["wlanEnable"],
					wifiSsid: data["wifiSsid"],
					wifiEncryption: data["wifiEncryption"],
					wifiPassphrase: data["wifiPassphrase"]
				});
				Object.assign(vm.form,{
					wifiSsid: data["wifiSsid"],
					wifiEncryption: data["wifiEncryption"],
					wifiPassphrase: data["wifiPassphrase"],
				}); 
				// sub 
				['1','2','3'].map(function(item){
					var row = {},pre = 'wifi',suf = 'Old',
						enableKey = pre+item+'Enable',
						idKey = pre+item+'Ssid',						
						encryKey = pre+item+'Encryption',
						phraKey = pre+item+'Passphrase';
					row['id'] = item;
					row['wifiEnable'] = data[enableKey];
					row['wifiSsid'] = data[idKey];
					row['wifiEncryption'] = data[encryKey];
					row['wifiPassphrase'] = data[phraKey];
					vm.form[enableKey] = data[enableKey];
					vm.form[idKey] = data[idKey];
					vm.form[encryKey] = data[encryKey];
					vm.form[phraKey] = data[phraKey];
					vm.tbData.push(row);
				}); 
			},
			//wifi5
			wifi5ReloadTable(){
				var vm = this;
				vm.tbDataWifi5.push({
					id: 'main',
					wifi5WifiEnable: '',
					wifi5WifiSsid: '',
					wifi5WifiEncryption: '',
					wifi5WifiPassphrase: ''
				}); 
				// Form更新
				Object.assign(vm.form,{
					wifi5WifiSsid: '',
					wifi5WifiEncryption: '',
					wifi5WifiPassphrase: ''
				});
				// sub 
				 ['1','2','3'].map(function(item){
					var row = {},pre = 'wifi5Wifi',
						enableKey = pre+item+'Enable',
						idKey = pre+item+'Ssid',
						encryKey = pre+item+'Encryption',
						phraKey = pre+item+'Passphrase';							
					
					row['id'] = item;
					row['wifi5WifiEnable'] = '';
					row['wifi5WifiSsid'] = '';
					row['wifi5WifiEncryption'] = '';
					row['wifi5WifiPassphrase'] = '';

					// Form更新
					vm.form[enableKey] = '';
					vm.form[idKey] = '';
					vm.form[encryKey] = '';
					vm.form[phraKey] = '';
					vm.tbDataWifi5.push(row); 
				}); 
			},
			wifi5RaloadTableData( data){
				var vm = this;
				vm.tbDataWifi5.push({
					id: 'main',
					wifi5WifiEnable: data["wifi5WlanEnable"],
					wifi5WifiSsid: data["wifi5WifiSsid"],
					wifi5WifiEncryption: data["wifi5WifiEncryption"],
					wifi5WifiPassphrase: data["wifi5WifiPassphrase"]
				});
				// Form更新
				Object.assign(vm.form,{
					wifi5WifiSsid: data["wifi5WifiSsid"],
					wifi5WifiEncryption: data["wifi5WifiEncryption"],
					wifi5WifiPassphrase: data["wifi5WifiPassphrase"],
				}); 
				// sub 
				['1','2','3'].map(function(item){
					var row = {},pre = 'wifi5Wifi',suf = 'Old',
						enableKey = pre+item+'Enable',
						idKey = pre+item+'Ssid',						
						encryKey = pre+item+'Encryption',
						phraKey = pre+item+'Passphrase';
					row['id'] = item;
					row['wifi5WifiEnable'] = data[enableKey];
					row['wifi5WifiSsid'] = data[idKey];
					row['wifi5WifiEncryption'] = data[encryKey];
					row['wifi5WifiPassphrase'] = data[phraKey];
					// Form更新
					vm.form[enableKey] = data[enableKey];
					vm.form[idKey] = data[idKey];
					vm.form[encryKey] = data[encryKey];
					vm.form[phraKey] = data[phraKey];
					vm.tbDataWifi5.push(row);
				}); 
			},
						
			/*确定导入*/
	        uploadParams() {
				var vm = this;
				vm.$refs.importRuleForm.validate((valid) => {
	                if (valid) {
	                	vm.$refs.upload.submit();                   	
	                }
	            }) 				
			},
			
			// 关闭导入弹出框
			closeImportParams(){
				var vm = this;			
				vm.fileList = [];
				vm.fileName = '';
				vm.$refs.importRuleForm.resetFields();
				vm.showImportCard = false;
			},
			//下载模板
			exportTemplate(){
				exportByForm('${ctx}/cpe/batchconfig/downloadTemplate.action', {});
			},
			
			//导出
			exportFileBtn(){
				var vm = this, newForm = vm.$refs.addform;
				var newParams = vm.formatParams(vm.form),
					pureParams = vm.pureParams(newForm,newParams);
				 exportByForm("${ctx}/cpe/batchconfig/exportBatchConfigurationInfos.action",pureParams);
			},
			// 导入 导出 end
						
			//add  earfcn
			addEarfcnBtn(){
				var vm = this , value = vm.form.onlyEarfcn;
				if(value && isNaN(value)) {
					vm.earfcnErrorMessage = '<%=rb.getString("PinDianGeShiCuoWu")%>';
				}else if(value) {
					if(value - 0 < 0 || value - 65535 > 0) {
						vm.earfcnErrorMessage = '<%=rb.getString("PinDianChaoChuFanWei")%>';
					}else {
						if(vm.form.earfcnList.indexOf(value) == -1){
							vm.form.earfcnList.push(value);
							vm.form.onlyEarfcn = '';
							vm.earfcnErrorMessage = '';
							vm.form.pci = vm.form.earfcnList.map(function(item){
								return item;							
							}).join(';');
						}else{
							vm.earfcnErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}
					}
				}else {
					vm.earfcnErrorMessage = '<%=rb.getString("PinDianWeiKong")%>';
				} 
			},
			//新建 earfcn and pci
			addPcilockBtn(){
				var vm = this , startValue = vm.form.earfcnStart, endValue = vm.form.pciEnd, regNum = /^\d+$/;
				if(startValue == '' || endValue == ''){
					vm.earfcnPciErrorMessage = '<%=rb.getString("QingShuRu")%><%=rb.getString("PinDian")%> , <%=rb.getString("PCI2")%>';
				}else{
					
					if(!regNum.test(startValue) || ( startValue - 0 < 0 || startValue - 65535 > 0)) {
						vm.earfcnPciErrorMessage = 'Earfcn <%=rb.getString("FanWei")%>：0 ~65535';
					}else if(!regNum.test(endValue) || (endValue - 0 < 0 || endValue-503 > 0)){
						vm.earfcnPciErrorMessage = 'PCI <%=rb.getString("FanWei")%>：0~503';
					}else{
						str = startValue + "," + endValue;
						if(vm.form.earfcnPciList.indexOf(str) == -1){
							vm.form.earfcnPciList.push({startValue,endValue});
							vm.form.earfcnStart = '';
							vm.form.pciEnd = '';
							vm.earfcnPciErrorMessage = '';
							vm.form.pci = vm.form.earfcnPciList.map(function(item){
								var curPci = item.startValue +','+ item.endValue;						
								return curPci;
							}).join(';');
						}else{
							vm.earfcnPciErrorMessage = '<%=rb.getString("IPFanWeiYiCunZai")%>';
						}  
					}					
				} 			
			},
			// add pci
			addPciBtn(){
				var vm = this , value = vm.form.onlyPci;
				if(value && isNaN(value)) {
					vm.pciErrorMessage = '<%=rb.getString("PCIGeShiCuoWu")%>';
				}else if(value) {
					if(value-0 < 0 || value-503 > 0) {
						vm.pciErrorMessage = '<%=rb.getString("PCIChaoChuFanWei")%>';
					}else {
						if(vm.form.pciList.indexOf(value) == -1){
							vm.form.pciList.push(value);
							vm.form.onlyPci = '';
							vm.pciErrorMessage = '';
							vm.form.pci = vm.form.pciList.map(function(item){
								return item;
							}).join(';');
						}else{
							vm.pciErrorMessage = '<%=rb.getString("PCIWeiKong")%>';
						}
					}
				}else {
					vm.pciErrorMessage = '<%=rb.getString("PCIWeiKong")%>';
				} 
			},
			removeEarfcn(item){
				var vm = this, index = vm.form.earfcnList.indexOf(item);
			
				if(index !== -1){
					vm.form.earfcnList.splice(index,1)
				}
				vm.form.pci = vm.form.earfcnList.map(function(item){
					return item;							
				}).join(';');
				vm.earfcnErrorMessage = '';
			},
			removeEarfcnPci(item){
				var vm = this, index = vm.form.earfcnPciList.indexOf(item);
				if(index !== -1){
					vm.form.earfcnPciList.splice(index,1)
				}
				vm.form.pci = vm.form.earfcnPciList.map(function(item){
					var curPci = item.startValue +','+ item.endValue;						
					return curPci;
				}).join(';');
				vm.earfcnPciErrorMessage = '';
			},
			removePci(item){
				var vm = this, index = vm.form.pciList.indexOf(item);
				if(index !== -1){
					vm.form.pciList.splice(index,1)
				}
				vm.form.pci = form.pciList.map(function(item){
					return item;
				}).join(';');
				vm.pciErrorMessage = '';
			},
			
			modifyWifi(row) {
				var vm = this;
				vm.wifiMDLShow = true;
				vm.$nextTick(function(){
					vm.$refs.modifyForm.resetFields();
					Object.assign(vm.mForm, row);
				});
			},
			saveModify() {
				var vm = this;
				vm.$refs.modifyForm.validate(function(valid){
					if(valid) {
						vm.tbData.map(function(item){
							if(item.id == vm.mForm.id) {
								if(vm.mForm.wifiEnable == '0'){
									//置空表格SSID 和 security Mode
									vm.mForm.wifiSsid = '';
									vm.mForm.wifiEncryption = '';
									vm.mForm.wifiPassphrase = '';
									vm.mForm.showPassword = false
								}
								Object.assign(item, vm.mForm);
							}
						});
						vm.wifiMDLShow = false;
					}
				});
			},
			//wifi5
			modifyWifi5(row) {
				var vm = this;

				vm.wifi5WifiMDLShow = true;
				vm.$nextTick(function(){
					vm.$refs.modifyWifi5Form.resetFields();
					Object.assign(vm.mWifi5Form, row);
				});
			},
			saveModifyWifi5() {
				var vm = this;

				vm.$refs.modifyWifi5Form.validate(function(valid, errors){
					if(valid) {
						vm.tbDataWifi5.map(function(item){
							if(item.id == vm.mWifi5Form.id) {
								if(vm.mWifi5Form.wifi5WifiEnable == '0'){
									//置空表格SSID 和 security Mode
									vm.mWifi5Form.wifi5WifiSsid = '';
									vm.mWifi5Form.wifi5WifiEncryption = '';
									vm.mWifi5Form.wifi5WifiPassphrase = '';
									vm.mWifi5Form.wifi5ShowPassword = false;
								}
								Object.assign(item, vm.mWifi5Form);
							}
						});
						vm.wifi5WifiMDLShow = false;
					}
				});
			},

        	handleSizeChange(val){
				this.pageSize = val;
			},
			handleCurrentChange(val){
				this.currentPage = val;
			},
			//NEW IP
        	systemAddIp(){			
				this.addIpShow = true;
			},
			//update ip
			updateIp(row,index){ 
				var vm = this;
				vm.editIndex = index;
				vm.editIpForm.ipStart = row.ipStart;
				vm.editIpForm.ipEnd = row.ipEnd;
				vm.oldEditIpStart = row.ipStart,
				vm.oldEditIpEnd = row.ipEnd;
				vm.editIpShow = true;
		    },
		    saveEditIp(){
		    	var vm = this,
					reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
					ipStart = vm.editIpForm.ipStart,
					ipEnd = vm.editIpForm.ipEnd;
		    	
				//endIp不是必填项
				if(ipStart === '' || ipStart === null || ipStart === undefined){					
					vm.editErrorMsg = '<%=rb.getString("IPDiZhiBuNengWeiKong")%>';
				}else{
					//如果只有start IP
					 if(ipEnd == '' || ipEnd == null){
						 if(reg.test(ipStart)){
							 str = ipStart;
							 if(ipStart == vm.oldEditIpStart){
								 vm.editErrorMsg = '<%=rb.getString("IPYiCunZai")%>';
							 }else{
								 vm.ipTableData.splice(vm.editIndex,1,vm.editIpForm)
								 vm.closeEditIp();
							 }
						 }else{
						 	 vm.editErrorMsg = '<%=rb.getString("IPDiZhiFeiFa")%>';
						 }
					}else{
						//如果star ip ，end ip都有
						 if(reg.test(ipStart) && reg.test(ipEnd) && vm.compareIp(ipStart,ipEnd)){
							 if(ipStart == vm.oldEditIpStart && ipEnd == vm.oldEditIpEnd){
								 vm.editErrorMsg = '<%=rb.getString("IPFanWeiYiCunZai")%>';
							 }else{
								 vm.ipTableData.splice(vm.editIndex,1,vm.editIpForm)
								 vm.closeEditIp();
							 }
						 }else{
							vm.editErrorMsg = '<%=rb.getString("IPDiZhiFeiFa")%>';
						 }
					 }											
					 vm.form.rangeIp = vm.ipTableData.map((item,index) => {
						if(item.ipEnd ){
							return item.ipStart + '-' + item.ipEnd;
						}else{
							return item.ipStart;
						}								
					}).join(';');
				} 
		    },
		    closeEditIp(){
		    	var vm = this;
		    	vm.editIpForm = {
					ipStart:'',
					ipEnd:''
				}
		    	vm.editErrorMsg = '';
				vm.$refs.editIpForm.resetFields();
		    	vm.editIpShow = false;
		    },
			//ip list 表格删除
			deleteIp(row,index){
				var vm = this;
				if(index >= 0) {
					vm.ipTableData.splice(index,1);			    	
			    }
				vm.form.rangeIp =  vm.ipTableData.map((item,index) => {
					if(item.ipEnd ){
						return item.ipStart + '-' + item.ipEnd;
					}else{
						return item.ipStart;
					}								
				}).join(';');
		    },
			
			//添加ip
			addIpBtn(){
		    	var vm = this,
					reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
					ipStart = vm.addIpForm.ipStart,
					ipEnd = vm.addIpForm.ipEnd,
					str = '';
				
				//endIp不是必填项
				if(ipStart == ''){					
					vm.errorMessage = '<%=rb.getString("IPDiZhiBuNengWeiKong")%>';
				}else{
					//如果只有start IP
					if(ipEnd == '' || ipEnd == null){
						if(reg.test(ipStart)){
							str = ipStart;
							if(vm.addIpForm.ipGroup.indexOf(str) == -1){
								vm.addIpForm.ipGroup.push(str);
								vm.addIpForm.ipStart = '';
								vm.errorMessage = '';
							}else{
								vm.errorMessage = '<%=rb.getString("IPYiCunZai")%>';
							}
						}else{
							vm.errorMessage = '<%=rb.getString("IPDiZhiFeiFa")%>';
						}
					}else{
						//如果star ip ，end ip都有
						if(reg.test(ipStart) && reg.test(ipEnd) && vm.compareIp(ipStart,ipEnd)){
							str = ipStart + "-" + ipEnd;
							if(vm.addIpForm.ipGroup.indexOf(str) == -1){
								vm.addIpForm.ipGroup.push(str);
								vm.addIpForm.ipStart = '';
								vm.addIpForm.ipEnd = '';
								vm.errorMessage = '';
							}else{
								vm.errorMessage = '<%=rb.getString("IPFanWeiYiCunZai")%>';
							}
						}else{
							vm.errorMessage = '<%=rb.getString("IPDiZhiFeiFa")%>';
						}
					}										
				} 
			},
			/**
			 * 新建ip弹窗中的单个删除ip
			 * @param item {string} 删除项
			*/
			removeIp(item){
				var vm = this, index = vm.addIpForm.ipGroup.indexOf(item);				
				if(index !== -1){
					vm.addIpForm.ipGroup.splice(index,1)
				}
				vm.errorMessage = '';
			},
			/**
			 * 比较ip大小
			 * @param ipStart {string} 初始ip
			 * @param ipEnd {string} 结束ip
			*/
			compareIp(ipStart,ipEnd){
				var temp1,
					temp2,
					bool = true;
				temp1 = ipStart.split(".");
				temp2 = ipEnd.split(".");
				for(var i=0;i<4;i++){
					if(parseInt(temp1[i])>parseInt(temp2[i])){
						bool = false;
					}
				}
				return bool;
			},
			
			//添加 ip
			saveAddIp(){
				var vm = this;
				if(vm.addIpForm.ipGroup.length == 0){
					vm.errorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
				}else{
					var ipListData = vm.addIpForm.ipGroup.toString().split(",").join(";"),									
						newIpListData = ipListData.split(';'),
						getRowData ={};				
					newIpListData.map((item,index) => {
						if(item.includes('-')){
							getRowData = {     						
								ipStart:item.split('-')[0],
	      						ipEnd:item.split('-')[1]	    			
	      		    		}
						}else{
							getRowData = {     						
								ipStart:item	    			
		      		    	}
						}
						vm.ipTableData.push(getRowData);
					}) 
					vm.oldNewIpTableData();
					vm.closeAddIp();
					vm.form.rangeIp = vm.ipTableData.map((item,index) => {
						if(item.ipEnd ){
							return item.ipStart + '-' + item.ipEnd;
						}else{
							return item.ipStart;
						}								
					}).join(';');
				}				
			},
			
			oldNewIpTableData(){
				var vm = this;
				for(var i = 0; i< vm.ipTableData.length; i++){
					for(var j = i+1; j < vm.ipTableData.length; j++ ){
						//两种情况，1：只有 start ip; 2 start ip, end ip 都有时
						if(vm.ipTableData[i].ipEnd){
							if(vm.ipTableData[i].ipStart == vm.ipTableData[j].ipStart && vm.ipTableData[i].ipEnd == vm.ipTableData[j].ipEnd){
								vm.ipTableData.splice(j,1);								
								j--;
							}
						}else{
							if(vm.ipTableData[i].ipStart == vm.ipTableData[j].ipStart){
								vm.ipTableData.splice(j,1);							
								j--;
							}
						}					
					}					
				}
				return vm.ipTableData;
			},
			
			//关闭添加IP弹窗
			closeAddIp(){
				var vm = this;
				vm.addIpForm = {
					ipStart:'',
					ipEnd:'',
					ipGroup:[]
				}
				vm.errorMessage = '';
				vm.$refs.addIpForm.resetFields();
				vm.addIpShow = false;			
			},
			
			//wifi 开关改变
			wlanEnableChange(val) {
				var vm = this,
				row = vm.tbData[0];
				if(row) {
					row.wifiEnable = val == '1'?'1':'0';
				}
			},						
			wifi5EnableChange(val) {
				var vm = this,
					row = vm.tbDataWifi5[0];
				
				if(row) {
					row.wifi5WifiEnable = val == '1'?'1':'0';
				}
			},
			loadSuccessDevice(){
				initForm(this.$refs.addform)
			},
			// 右侧的筛选搜索
			queryDeviceList(val){
				var vm = this;
				vm.queryParams.searchText = val;
			},

			/**
			 * 设备选择变化时，更新选择设备记录
			 * @param value:
			*/
			devicesChange(value) {
				var vm = this;
				vm.$nextTick(function(){
					var rows = vm.$refs.deviceListPairgrid.getData();
					vm.form.cpeCodes = rows.map(function(row){ return row.small_cell_code ;}).sort().join(',');
				})
			},
			
			// 关闭
			closePanel(){
				var vm = this;
				if(vm.operateType == 'information'){
					eventBus.$emit('hide-slide');
				}else{
					vm.cancel();
				}				
			}, 

			// 提交
			formSubmit(){ 
				var vm = this,			
					params = {},
					newForm = vm.$refs.addform;
					message = '<%=rb.getString("ChengGong")%>';
                // 防止多次提交
                if(vm.submitLoading)return
				//params 必传参数
				params.timeZone = timeZone;
				params.taskName = vm.form.taskName;
				params.time = '';
				params.executeType = 'active';
				params.taskId = '';
				//1 all，此时 cpeCodes 参数为 空, 0:指定，cpeCodes 以逗号分隔；
				if(vm.selectAll == '0'){
					params.selectAll = '0';
					params.cpeCodes = vm.form.cpeCodes;
				}else{
					params.selectAll = '1';
					params.cpeCodes = '';
				}
				
				var flagEarfcn = false;	
				//earfcn, earfcn and pci, pci
				if(vm.cpeModel == '4G'){
					if(vm.form.scanMode == 'freqpreferred'){
						if(vm.form.earfcnList.length == 0){
							flagEarfcn = true;
							vm.earfcnErrorMessage = '<%=rb.getString("PinDianWeiKong")%>';
						}else{
							flagEarfcn = false;
							vm.earfcnErrorMessage = '';						
						} 
					}else if(vm.form.scanMode == 'pcilock'){
						if(vm.form.earfcnPciList.length == 0){
							flagEarfcn = true;
							vm.earfcnPciErrorMessage = '<%=rb.getString("PinDianWeiKong")%> ';
						}else{
							flagEarfcn = false;
							vm.earfcnPciErrorMessage = '';						
						}
					}else if(vm.form.scanMode == 'pcionlylock'){
						if(vm.form.pciList.length == 0){
							flagEarfcn = true;
							vm.pciErrorMessage = '<%=rb.getString("PCIWeiKong")%>';
						}else{
							flagEarfcn = false;
							vm.pciErrorMessage = '';						
						}
					}
				}
				vm.$refs.addform.validate(function(valid){
					if(valid && flagEarfcn == false){
						var newParams = vm.formatParams(vm.form),
							pureParams = vm.pureParams(newForm,newParams);
						Object.assign(params, pureParams);
						if(isFormChanged(vm.$refs.addform)) {
                            vm.submitLoading = true;
							axios.post('${ctx}/cpe/batchconfig/addTask.action',stringify(params)).then(function(response){
								var data = response.data;
			    				if(data["success"]){
			    					vm.$message({
			    						message: message,
			    						type:'success',
			    					})
                                    eventBus.$emit('hide-slide');
			    				}else{
			    					vm.$message.error(data["message"]);
                                    vm.submitLoading = false;
			    				}
							}).catch(function(error){})
						}else{
							vm.$message({
								message: '<%=rb.getString("CanShuZhiMeiYouBianHua")%>',
								type: 'warning'
							})
                            
						}						
					}
				});
			},
			
			pureParams(newForm,params) {
				var vm = this, 
					map = {}, 
					paramConfig = {},
					changeKeys = [],
					isMacChange = false,
					isIpChange = false,
					isUrlChange = false;
				newForm.fields.map(function(field){
					//根据prop  找对应的组					
					var groups ={							
							wlanEnable: 'wifi',
							wifiChannel: 'wifi',
							wifiBandwidth: 'wifi',							
							wifiSsid: 'wifi',
							wifiEncryption: 'wifi',
							wifiPassphrase: 'wifi',
							wifi1Enable: 'wifi',
							wifi1Ssid: 'wifi',
							wifi1Encryption: 'wifi',
							wifi1Passphrase: 'wifi',						
							wifi2Enable: 'wifi',
							wifi2Ssid: 'wifi',
							wifi2Encryption: 'wifi',
							wifi2Passphrase: 'wifi',							
							wifi3Enable: 'wifi',
							wifi3Ssid: 'wifi',
							wifi3Encryption: 'wifi',
							wifi3Passphrase: 'wifi',
							//wifi5
							wifi5WlanEnable: 'wifi5', //wifi5 enable
							wifi5WifiMode: 'wifi5', //network mode
							wifi5WifiChannel: 'wifi5',//channel  
							wifi5WifiBandwidth: 'wifi5',//channel bandwith
							wifi5WifiSupportChannel: 'wifi5',//support channel 
							
							wifi5WifiSsid: 'wifi5',
							wifi5WifiEncryption: 'wifi5',
							wifi5WifiPassphrase: 'wifi5',
		
							wifi5Wifi1Enable: 'wifi5',
							wifi5Wifi1Ssid: 'wifi5',
							wifi5Wifi1Encryption: 'wifi5',
							wifi5Wifi1Passphrase: 'wifi5',
							
							wifi5Wifi2Enable: 'wifi5',
							wifi5Wifi2Ssid: 'wifi5',
							wifi5Wifi2Encryption: 'wifi5',
							wifi5Wifi2Passphrase: 'wifi5',
							
							wifi5Wifi3Enable: 'wifi5',
							wifi5Wifi3Ssid: 'wifi5',
							wifi5Wifi3Encryption: 'wifi5',
							wifi5Wifi3Passphrase: 'wifi5',
							
							dmzEnable:'dmz',
							dmzHostAddress:'dmz',						
							lanEnable: 'lan',							
							scanMode: 'pciLock',
							pci: 'pciLock',
							uiPassword: 'wanAcc',
							httpsEnable: 'wanAcc',
							httpsWanEnable: 'wanAcc',
							wanAccEnable: 'wanAcc',
							rangeIp: 'wanAcc',							
							watchDogEnable: 'watchDog',
							watchDogPingIp: 'watchDog',
							watchDogPingTimeout: 'watchDog',
							watchDogPingCount: 'watchDog',
							watchDogFailureReboot: 'watchDog',
							
							snmpEnable: 'snmp',
							nmsAddress: 'snmp',
							nmsPort: 'snmp',
							listeningPort: 'snmp',
							trapCommunity: 'snmp',
							version: 'snmp',
							readCommunity: 'snmp',
							rwCommunity: 'snmp',
							userName: 'snmp',
							authenticationProtocol: 'snmp',
							authenticationPassphrase: 'snmp',
							privacyProtocol: 'snmp',
							privacyPassphrase: 'snmp'
						},
						prop = field.prop,
						groupName = groups[prop];
					if(Array.isArray(field.fieldValue)){
						var vList = field.fieldValue.map(function(item){return item}),
							oList = (field.reinitialValue||[]).map(function(item){return item}),
							val = JSON.stringify(vList.sort()),
							orVal = JSON.stringify(oList.sort());
	
						if(val != orVal) {
							changeKeys.push(field.prop);
						};
					}else{
						if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
							
						}else if(field.fieldValue != field.reinitialValue) {
							var key = field.prop;
							changeKeys.push(field.prop);
						};
					}
					if(groupName === undefined){
						var basicSettingKeys = ['macFilterEnable','macFilterMode','macFilterWhiteList','macFilterBlackList','ipFilterEnable','ipFilterMode','ipFilterWhiteList','ipFilterBlackList',
						'urlFilterEnable','urlFilterMode','urlFilterBlackList'];
						if(basicSettingKeys.indexOf(prop) > -1){
							
						}else{
							if(field.fieldValue !== field.reinitialValue) map[prop] = field.fieldValue;	
						}
					}
					if(field.fieldValue !== field.reinitialValue && groupName !== undefined) {
						if(paramConfig[groupName] === undefined ){
							paramConfig[groupName] = {};						
						}
						paramConfig[groupName][prop] = field.fieldValue; 
					}; 
				});
				['macFilterEnable','macFilterMode','macFilterWhiteList','macFilterBlackList'].map(function(key){
					if(changeKeys.indexOf(key) != -1){
						isMacChange = true;
					}
				});
				['ipFilterEnable','ipFilterMode','ipFilterWhiteList','ipFilterBlackList'].map(function(key){
					if(changeKeys.indexOf(key) != -1){
						isIpChange = true;
					}
				});
				['urlFilterEnable','urlFilterMode','urlFilterBlackList'].map(function(key){
					if(changeKeys.indexOf(key) != -1){
						isUrlChange = true;
					}
				});
				if(isMacChange){
					var macFilter={};
						macFilterWhiteList = vm.form.macFilterWhiteList.map(function(item){return item.macAddress}),
						macFilterBlackList = vm.form.macFilterBlackList.map(function(item){return item.macAddress});
					['macFilterEnable','macFilterMode'].map((key,index)=>{
						macFilter[key] = vm.form[key];
					});
					macFilter.macFilterWhiteList = macFilterWhiteList.join(';');
					macFilter.macFilterBlackList = macFilterBlackList.join(';');
					paramConfig['macFilter'] = macFilter;
				}
				if(isIpChange){
					var ipFilter={},
						ipFilterWhiteList = vm.form.ipFilterWhiteList.map(function(item){return item.sourceIp}),
						ipFilterBlackList = vm.form.ipFilterBlackList.map(function(item){return item.sourceIp});
					['ipFilterEnable','ipFilterMode'].map((key,index)=>{
						ipFilter[key] = vm.form[key];
					});
					ipFilter.ipFilterWhiteList = ipFilterWhiteList.join(';');
					ipFilter.ipFilterBlackList = ipFilterBlackList.join(';');
					paramConfig['ipFilter'] = ipFilter;
				}
				if(isUrlChange){
					var urlFilter={},
						urlFilterBlackList = vm.form.urlFilterBlackList.map(function(item){return item.url});
					['urlFilterEnable','urlFilterMode'].map((key,index)=>{
						urlFilter[key] = vm.form[key];
					});
					urlFilter.urlFilterBlackList = urlFilterBlackList.join(';');
					paramConfig['urlFilter'] = urlFilter;
				}
				if(vm.cpeModel == '4G'){
					paramConfig.cpeType = '4';
				}else{
					paramConfig.cpeType = '5';
				}
				map.paramConfig = JSON.stringify(paramConfig);
				return map;
			},
			
			formatParams(form) {
				var vm = this,
					code = vm.activeCode,
					params = {};
							
					params.rangeIp = vm.ipTableData.map((item,index) => {
						if(item.ipEnd ){
							return item.ipStart + '-' + item.ipEnd;
						}else{
							return item.ipStart;
						}								
					}).join(';');

					var scanMode = form.scanMode;
					params.scanMode = scanMode;
					if(vm.cpeModel == '4G'){
						//earfcn
						if(scanMode == 'freqpreferred') {
							params.pci = form.earfcnList.map(function(item){
								return item;							
							}).join(';');						
						}
						// earfcn and pci
						if(scanMode == 'pcilock') {
							params.pci = form.earfcnPciList.map(function(item){
								var curPci = item.startValue +','+ item.endValue;						
								return curPci;
							}).join(';');
						}
						//pci
						if(scanMode == 'pcionlylock') {
							params.pci = form.pciList.map(function(item){
								return item;
							}).join(';');
						}
					}else{
						if(scanMode == 'celllock') {
							params['pci'] = vm.cellLockList.map(function(row){
								var curPci = row.rat +','+ row.band +','+ row.earfcn +','+ row.pci;
								return curPci;
							}).join(';');
						}else if(scanMode == 'bandlock') {
							params['pci'] = vm.bandLockList.map(function(row){
								var curPci = row.rat +','+ row.band;
								return curPci;
							}).join(';');
						}else if(scanMode == 'freqlock'){
							params['pci']= vm.freqLockList.map(function(row){
								var curPci = row.rat +','+ row.band +','+ row.freq
								return curPci;
							}).join(';');
						}
					}
					vm.tbData.map(function(row){
						if(['1','2','3'].includes(row.id)) {
							var index = row.id,
								pre = 'wifi',
								enableKey = pre+index+'Enable',
								idKey = pre+index+'Ssid',
								encryKey = pre+index+'Encryption',
								phraKey = pre+index+'Passphrase';							
							form[enableKey] = row['wifiEnable'];
							form[idKey] = row['wifiSsid'];
							form[encryKey] = row['wifiEncryption'];
							form[phraKey] = row['wifiPassphrase'];
						}else {
							Object.assign(form,{
								wifiSsid: row['wifiSsid'],
								wifiEncryption: row['wifiEncryption'],
								wifiPassphrase: row['wifiPassphrase']
							});
						}
					});
					//wifi5
					vm.tbDataWifi5.map(function(row){
						if(['1','2','3'].includes(row.id)) {
							var index = row.id,
								pre = 'wifi5Wifi',
								enableKey = pre+index+'Enable',
								idKey = pre+index+'Ssid',
								encryKey = pre+index+'Encryption',
								phraKey = pre+index+'Passphrase';							
							form[enableKey] = row['wifi5WifiEnable'];
							form[idKey] = row['wifi5WifiSsid'];
							form[encryKey] = row['wifi5WifiEncryption'];
							form[phraKey] = row['wifi5WifiPassphrase'];
						}else {
							Object.assign(form,{
								wifi5WifiSsid: row['wifi5WifiSsid'],
								wifi5WifiEncryption: row['wifi5WifiEncryption'],
								wifi5WifiPassphrase: row['wifi5WifiPassphrase']
							});
						}
					});
				
					if(form.wlanEnable == '1') {
						Object.assign(params, form);
					}else {
						Object.assign(params, {
							wlanEnable: form.wlanEnable
						});
					}	
					if(form.wifi5WlanEnable == '1') {
						Object.assign(params, form);
					}else {
						Object.assign(params, {
							wifi5WlanEnable: form.wifi5WlanEnable
						});
					}		
				return params;
			},
			
			// 关闭新建弹窗
			cancel(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>';
				if(isFormChanged(vm.$refs.addform)){
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						customClass:'warningConfirm',
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						eventBus.$emit('hide-slide')
					}).catch(() => {
						
					})
				}else{
					eventBus.$emit('hide-slide')
				}
			},
		
			// 设备执行类别  1 全部执行 0 指定执行
			selectAllChange(val){
				var vm = this;				
				vm.queryParams.searchText = '';
				if(val == '1'){
					vm.rightUrl = '';
				}else{
					vm.$refs.addform.validateField('cpeCodes');
					if(vm.operateType == 'information'){
						vm.rightUrl="${ctx}/cpe/batchconfig/getSelectedCPEInfo.action?taskId="+vm.taskId;
					}					
				}
			},
			// Mac Address添加事件
			macAddressAdd(){
				var vm = this,
					val = vm.form.macAddress,
					addType = vm.form.macAddMode,
					filterType = vm.form.macFilterMode,
					listCode = {
						'1':'macFilterWhiteList',
						'0':'macFilterBlackList'
					},
					params={
						macAddress:vm.form.macAddress,
					};
				if(vm.form[listCode[filterType]].length >= 128){
					vm.$message.warning('No more than 128');
					return;
				}
				if(addType == '0'){
					if(val){
						if(vm.isValidMacAddress(val)) {
							var result = vm.form[listCode[filterType]].some(item=>item.macAddress == val);
							if(result){
								vm.macAddressErrorMessage = '<%=rb.getString("YiCunZai")%>';
							}else{
								vm.form[listCode[filterType]].push(params);
								vm.form.macAddress = '';
								vm.macAddressErrorMessage = '';
							}
						}else {
							vm.macAddressErrorMessage = 'Format：xx:xx:xx:xx:xx:xx,no more than 128';
						}
					}
				}
			},
			// Mac Address  删除事件
			macAddressListDel(row){
				var vm = this
					filterType = vm.form.macFilterMode,
					listCode = {
						'1':'macFilterWhiteList',
						'0':'macFilterBlackList'
					};
				vm.form[listCode[filterType]] = vm.form[listCode[filterType]].filter((items)=>{
					return items.macAddress != row.macAddress
				})
			},
			macAddressAddBeforeUpload(file){
				var vm = this, 
					urls = '${ctx}/cell/CPE/setting/importMacFilterInfo.action',
					FileName = file.name,
					fd = new FormData(),
					config = {
						headers: { 'Content-Type': 'multipart/form-data' }
					};
				fd.append('uploadFile',file); //文件流
				fd.append('FileName',FileName);//文件名
				axios.post(urls,fd,config).then(function(res){
					let data = res.data;
					if(data['success']){
						if(data.macList){
							var macList = data.macList.split(';'),
								filterType = vm.form.macFilterMode,
								listCode = {
									'1':'macFilterWhiteList',
									'0':'macFilterBlackList'
								};
							try{
								macList.forEach(item=>{
									let isExist = vm.form[listCode[filterType]].some(items=>items.macAddress == item);
									
									if(!isExist){
										if(vm.form[listCode[filterType]].length >= 128){
											vm.$message.warning('<%=rb.getString("PiLiangDaoRuChaoXianTiShi")%>');
											throw Error();
										}else{
											vm.form[listCode[filterType]].push({
												macAddress:item
											})
										}
									}
								})
							}catch(e){}
							
						}else{
							vm.$message.warning('No valid data in the file');
						}
						vm.macAddressCloseAddFileSelect();
					}else{
						vm.$message.error(data["msg"])
					}
				})
				return false;
			},
			macAddressAddCheckFile(res,file){    //发送请求，校验device文件内容 
				var vm = this;
				if(res.success){
					if(res.suc_count>0){
						vm.$message({
							type: 'success',
							message: '<%=rb.getString("ChengGong")%>'
						});
					}else {
						vm.$message({
							type: 'warning',
							message: '<%=rb.getString("ShiBai")%>'
						});
					}
					vm.macAddressCloseAddFileSelect();
				}else{
					vm.$message({
						type: 'error',
						message: res.msg
					});
				}
				//修改已选择文件状态  
				var fileList = vm.$refs.macAddressAddUpload.uploadFiles;
				fileList.forEach(function(file){
					file.status = 'ready';
				})
			},
			// 移除导入文件
			macAddressCloseAddFileSelect(){
				var vm = this;
				vm.macAddressAddFileName = '';
				vm.$refs.macAddressAddUpload.clearFiles();
			},
			// 选择文件
			macAddressAddFileSelect(){  
				var vm =this;
				vm.$refs.macAddressAddUpload.clearFiles();
				vm.$refs['macAddressAddFile_up'].click();
			},
			/**
			* 选择文件后，校验格式，并赋值页面显示 
			* @param file{object}   文件信息
			* @param fileList{Array}  文件列表
			*/ 
			macAddressAddFileChange(file,fileList){ 
				var vm = this;
				const typeFlag = file.name.substr(file.name.lastIndexOf("."))  === '.xlsx' || file.name.substr(file.name.lastIndexOf("."))  === '.xls'
				
				if(typeFlag){
					vm.macAddressAddFileName = file.name;
					vm.macAddressErrorMessage = '';
					if(vm.macAddressAddFileName){
						vm.$refs.macAddressAddUpload.submit();
					}		
				}else {
					vm.macAddressAddFileName = '';
					vm.macAddressErrorMessage = '<%=rb.getString("DangQianZhiChiWenJianLeiXing")%>'
				}
			},
			// 导出 Mac List
			macAddressListExportClick(){ 
				var vm = this,
					urls = '${ctx}/cell/CPE/setting/exportMacFilterInfo.action',
					filterType = vm.form.macFilterMode,
					listCode = {
						'1':'macFilterWhiteList',
						'0':'macFilterBlackList'
					},
					params = {
						macs:''
					};
				params.macs = vm.form[listCode[filterType]].map(item=>item.macAddress).join(';');
				var bool = checkParams(params);
				if(!bool) return false;
				exportByForm(urls,params)
			},
			// source Ip添加事件
			sourceIpAdd(){
				var vm = this,
					val = vm.form.sourceIp,
					addType = vm.form.ipAddMode,
					filterType = vm.form.ipFilterMode,
					listCode = {
						'1':'ipFilterWhiteList',
						'0':'ipFilterBlackList'
					}
					params={
						sourceIp:vm.form.sourceIp,
					};
				if(vm.form[listCode[filterType]].length >= 128){
					vm.$message.warning('No more than 128');
					return;
				}
				if(addType == '0'){
					if(val){
						if(vm.isValidIP(val) || vm.isIPv6(val)) {
							var result = vm.form[listCode[filterType]].some(item=>item.sourceIp == val);
							if(result){
								vm.sourceIpErrorMessage = '<%=rb.getString("YiCunZai")%>';
							}else{
								vm.form[listCode[filterType]].push(params);
								vm.form.sourceIp = '';
								vm.sourceIpErrorMessage = '';
							}
						}else {
							vm.sourceIpErrorMessage = 'Support IPv4 or IPv6,no more than 128';
						}
					}
				}
			},
			// source Ip  删除事件
			sourceIpListDel(row){
				var vm = this
					filterType = vm.form.ipFilterMode,
					listCode = {
						'1':'ipFilterWhiteList',
						'0':'ipFilterBlackList'
					};
				vm.form[listCode[filterType]] = vm.form[listCode[filterType]].filter((items)=>{
					return items.sourceIp != row.sourceIp
				})
			},
			sourceIpAddBeforeUpload(file){
				var vm = this, 
					urls = '${ctx}/cell/CPE/setting/importIPFilterInfo.action',
					FileName = file.name,
					fd = new FormData(),
					config = {
						headers: { 'Content-Type': 'multipart/form-data' }
					};
				fd.append('uploadFile',file); //文件流
				fd.append('FileName',FileName);//文件名
				axios.post(urls,fd,config).then(function(res){
					let data = res.data;
					if(data['success']){
						if(data.ipList){
							var ipList = data.ipList.split(';'),
								filterType = vm.form.ipFilterMode,
								listCode = {
									'1':'ipFilterWhiteList',
									'0':'ipFilterBlackList'
								};
							try{
								ipList.forEach(item=>{
									let isExist = vm.form[listCode[filterType]].some(items=>items.sourceIp == item);
									
									if(!isExist){
										if(vm.form[listCode[filterType]].length >= 128){
											vm.$message.warning('<%=rb.getString("PiLiangDaoRuChaoXianTiShi")%>');
											throw Error();
										}else{
											vm.form[listCode[filterType]].push({
												sourceIp:item
											})
										}
									}
								})
							}catch(e){}
						}else{
							vm.$message.warning('No valid data in the file');
						}
						vm.sourceIpCloseAddFileSelect();
					}else{
						vm.$message.error(data["msg"])
					}
				})
				return false;
			},
			sourceIpAddCheckFile(res,file){    //发送请求，校验device文件内容 
				var vm = this;
				if(res.success){
					if(res.suc_count>0){
						vm.$message({
							type: 'success',
							message: '<%=rb.getString("ChengGong")%>'
						});
					}else {
						vm.$message({
							type: 'warning',
							message: '<%=rb.getString("ShiBai")%>'
						});
					}
					vm.sourceIpCloseAddFileSelect();
				}else{
					vm.$message({
						type: 'error',
						message: res.msg
					});
				}
				//修改已选择文件状态  
				var fileList = vm.$refs.sourceIpAddUpload.uploadFiles;
				fileList.forEach(function(file){
					file.status = 'ready';
				})
			},
			// 移除导入文件
			sourceIpCloseAddFileSelect(){
				var vm = this;
				vm.sourceIpAddFileName = '';
				vm.$refs.sourceIpAddUpload.clearFiles();
			},
			// 选择文件
			sourceIpAddFileSelect(){  
				var vm =this;
				vm.$refs.sourceIpAddUpload.clearFiles();
				vm.$refs['sourceIpAddFile_up'].click();
			},
			/**
			* 选择文件后，校验格式，并赋值页面显示 
			* @param file{object}   文件信息
			* @param fileList{Array}  文件列表
			*/ 
			sourceIpAddFileChange(file,fileList){ 
				var vm = this;
				const typeFlag = file.name.substr(file.name.lastIndexOf("."))  === '.xlsx' || file.name.substr(file.name.lastIndexOf("."))  === '.xls'
				
				if(typeFlag){
					vm.sourceIpAddFileName = file.name;
					vm.sourceIpErrorMessage = '';	
					if(vm.sourceIpAddFileName){
						vm.$refs.sourceIpAddUpload.submit();
					}			
				}else {
					vm.sourceIpAddFileName = '';
					vm.sourceIpErrorMessage = '<%=rb.getString("DangQianZhiChiWenJianLeiXing")%>'
				}
			},
			// 导出 source Ip List
			sourceIpListExportClick(){ 
				var vm = this,
					urls = '${ctx}/cell/CPE/setting/exportIPFilterInfo.action',
					filterType = vm.form.macFilterMode,
					listCode = {
						'1':'ipFilterWhiteList',
						'0':'ipFilterBlackList'
					},
					params = {
						ips:''
					};
				params.ips = vm.form[listCode[filterType]].map(item=>item.sourceIp).join(';');
				var bool = checkParams(params);
				if(!bool) return false;
				exportByForm(urls,params)
			},
			// URL 添加事件
			urlAdd(){
				var vm = this,
					val = vm.form.url,
					addType = vm.form.urlAddMode,
					filterType = vm.form.urlFilterMode,
					listCode = {
						'1':'urlFilterWhiteList',
						'0':'urlFilterBlackList'
					},
					params={
						url:vm.form.url,
					};
				if(vm.form[listCode[filterType]].length >= 64){
					vm.$message.warning('No more than 64');
					return;
				}
				if(addType == '0'){
					if(val){
						var result = vm.form[listCode[filterType]].some(item=>item.url == val);
						if(result){
							vm.urlErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}else{
							vm.form[listCode[filterType]].push(params);
							vm.form.url = '';
							vm.urlErrorMessage = '';
						}
					}
				}
			},
			// URL 删除事件
			urlListDel(row){
				var vm = this
					filterType = vm.form.urlFilterMode,
					listCode = {
						'1':'urlFilterWhiteList',
						'0':'urlFilterBlackList'
					};
				vm.form[listCode[filterType]] = vm.form[listCode[filterType]].filter((items)=>{
					return items.url != row.url
				})
			},
			urlAddBeforeUpload(file){
				var vm = this, 
					urls = '${ctx}/cell/CPE/setting/importIPFilterInfo.action',
					FileName = file.name,
					fd = new FormData(),
					config = {
						headers: { 'Content-Type': 'multipart/form-data' }
					};
				fd.append('uploadFile',file); //文件流
				fd.append('FileName',FileName);//文件名
				axios.post(urls,fd,config).then(function(res){
					let data = res.data;
					if(data['success']){
						if(data.urlList){
							var urlList = data.urlList.split(';'),
								filterType = vm.form.urlFilterMode,
								listCode = {
									'1':'urlFilterWhiteList',
									'0':'urlFilterBlackList'
								};
							try{
								urlList.forEach(item=>{
									let isExist = vm.form[listCode[filterType]].some(items=>items.url == item);
									
									if(!isExist){
										if(vm.form[listCode[filterType]].length >= 64){
											vm.$message.warning('<%=rb.getString("PiLiangDaoRuChaoXianTiShi")%>');
											throw Error();
										}else{
											vm.form[listCode[filterType]].push({
												url:item
											})
										}
									}
								})
							}catch(e){}
						}else{
							vm.$message.warning('No valid data in the file');
						}
						vm.urlCloseAddFileSelect();
					}else{
						vm.$message.error(data["msg"])
					}
				})
				return false;
			},
			urlAddCheckFile(res,file){    //发送请求，校验device文件内容 
				var vm = this;
				if(res.success){
					if(res.suc_count>0){
						vm.$message({
							type: 'success',
							message: '<%=rb.getString("ChengGong")%>'
						});
					}else {
						vm.$message({
							type: 'warning',
							message: '<%=rb.getString("ShiBai")%>'
						});
					}
					vm.urlCloseAddFileSelect();
				}else{
					vm.$message({
						type: 'error',
						message: res.msg
					});
				}
				//修改已选择文件状态  
				var fileList = vm.$refs.urlAddUpload.uploadFiles;
				fileList.forEach(function(file){
					file.status = 'ready';
				})
			},
			// 移除导入文件
			urlCloseAddFileSelect(){
				var vm = this;
				vm.urlAddFileName = '';
				vm.$refs.urlAddUpload.clearFiles();
			},
			// 选择文件
			urlAddFileSelect(){  
				var vm =this;
				vm.$refs.urlAddUpload.clearFiles();
				vm.$refs['urlAddFile_up'].click();
			},
			/**
			* 选择文件后，校验格式，并赋值页面显示 
			* @param file{object}   文件信息
			* @param fileList{Array}  文件列表
			*/ 
			urlAddFileChange(file,fileList){ 
				var vm = this;
				const typeFlag = file.name.substr(file.name.lastIndexOf("."))  === '.xlsx' || file.name.substr(file.name.lastIndexOf("."))  === '.xls'
				
				if(typeFlag){
					vm.urlAddFileName = file.name;
					vm.urlErrorMessage = '';
					if(vm.urlAddFileName){
						vm.$refs.urlAddUpload.submit();
					}				
				}else {
					vm.urlAddFileName = '';
					vm.urlErrorMessage = '<%=rb.getString("DangQianZhiChiWenJianLeiXing")%>'
				}
			},
			// 导出 URL List
			urlListExportClick(){ 
				var vm = this,
					urls = '${ctx}/cell/CPE/setting/exportUrlFilterInfo.action',
					filterType = vm.form.urlFilterMode,
					listCode = {
						'1':'urlFilterWhiteList',
						'0':'urlFilterBlackList'
					},
					params = {
						urls:''
					};
				params.urls = vm.form[listCode[filterType]].map(item=>item.url).join(';');
				var bool = checkParams(params);
				if(!bool) return false;
				exportByForm(urls,params)
			},
			// 导出模板
			exportAddTemplate(type){
				var vm = this,
					urlCodes={
						'MAC':'${ctx}/cell/CPE/setting/exportMacFilterTemplateInfo.action',
						'IP':'${ctx}/cell/CPE/setting/exportIPFilterTemplateInfo.action',
						'URL':'${ctx}/cell/CPE/setting/exportUrlFilterTemplateInfo.action'
					}
					urls = urlCodes[type],
					params = {};
				
				var bool = checkParams(params);
				if(!bool) return false;
				exportByForm(urls,params)
			},
			// 判断是否为空
			isNull(val){
				if(val==undefined || val == null || val =="") return true;
				else return false;
			},
			//校验MAC
			isValidMacAddress(mac){
				var reg = /^[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}$/;
				return reg.test(mac); 
			},
			//校验IP
			isValidIP(ip){
				var reg =  /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/     
				return reg.test(ip);     
			},
			//Ipv6校验 
			isIPv6(str){ 
				var reg = /^([\da-fA-F]{1,4}:){6}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^::([\da-fA-F]{1,4}:){0,4}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:):([\da-fA-F]{1,4}:){0,3}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){2}:([\da-fA-F]{1,4}:){0,2}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){3}:([\da-fA-F]{1,4}:){0,1}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){4}:((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){7}[\da-fA-F]{1,4}$|^:((:[\da-fA-F]{1,4}){1,6}|:)$|^[\da-fA-F]{1,4}:((:[\da-fA-F]{1,4}){1,5}|:)$|^([\da-fA-F]{1,4}:){2}((:[\da-fA-F]{1,4}){1,4}|:)$|^([\da-fA-F]{1,4}:){3}((:[\da-fA-F]{1,4}){1,3}|:)$|^([\da-fA-F]{1,4}:){4}((:[\da-fA-F]{1,4}){1,2}|:)$|^([\da-fA-F]{1,4}:){5}:([\da-fA-F]{1,4})?$|^([\da-fA-F]{1,4}:){6}:$/
				return reg.test(str);
			},
			//校验子网掩码
			isMask(str){
				var exp=/^(254|252|248|240|224|192|128|0)\.0\.0\.0|255\.(254|252|248|240|224|192|128|0)\.0\.0|255\.255\.(254|252|248|240|224|192|128|0)\.0|255\.255\.255\.(254|252|248|240|224|192|128|0)$/; 
				return exp.test(str); 		
			},
			// 验证输入的是否是数字
			isNumeric(str) {
				if(str.length==0){
					return false;
				}
				for(var i=0;i<str.length;i++){
					if(str.charAt(i)<"0" || str.charAt(i)>"9"){
						return false;
					}
				}
				return true;  
        	},
			// 5G 锁频 方式改变事件
			scanModeChange(){
				var vm = this;
				vm.form.pci = '';
			},
			showFreqLock() {// 5G 锁频 FreqLock
				let vm = this;
				vm.resetLockForm();
				vm.freq5gDlShow = true;
				vm.$nextTick(function(){
					vm.$refs.freqForm.clearValidate();
				});
			},
			showCellLock() {// 5G 锁频 CellLock
				let vm = this;
				vm.resetLockForm();
				vm.cell5gDlShow = true;
				vm.$nextTick(function(){
					vm.$refs.cellForm.clearValidate();
				});
			},
			showBandLock() {// 5G 锁频 BandLock
				let vm = this;
				vm.resetLockForm();
				vm.band5gDlShow = true;
				vm.$nextTick(function(){
					vm.$refs.bandForm.clearValidate();
				});
			},
			resetLockForm() { // 重置锁频表单
				var vm = this;
				Object.assign(vm.freqForm,{
					rat: '0',
					band: '1',
					freq: '',
				});
				Object.assign(vm.cellForm,{
					rat: '',
					band: '',
					earfcn: '',
					pci: ''
				});
				Object.assign(vm.bandForm,{
					rat: '',
					band: ''
				});
			},
			addFreqLock() { // 新增 5G 锁频 FreqLock
				var vm = this;
				var freqLockList = vm.freqLockList;
				var nrList = [],lteList = [];
				freqLockList.map((item)=>{
					if(item.rat == '0'){
						lteList.push(item)
					}else{
						nrList.push(item)
					}
				})
				if((lteList.length == 2 && vm.freqForm.rat == '0') || (nrList.length == 10 && vm.freqForm.rat == '1')){
					vm.$message.warning('5G NR：No more than 10,4G LTE：No more than 2')
					return
				}
				vm.$refs.freqForm.validate(function(r){
					if(r) {
						vm.freqLockList.push(Object.assign({},vm.freqForm));
						vm.freq5gDlShow = false;
					}
				});
			},
			addCellLock() { // 新增 5G 锁频 CellLock
				var vm = this;

				vm.$refs.cellForm.validate(function(r){
					if(r) {
						vm.cellLockList.push(Object.assign({},vm.cellForm));
						vm.cell5gDlShow = false;
					}
				});
			},
			addBandLock() { // 新增 5G 锁频 BandLock
				var vm = this;
				
				vm.$refs.bandForm.validate(function(r){
					if(r) {
						vm.bandLockList.push(Object.assign({},vm.bandForm));
						vm.band5gDlShow = false;
					}
				});
			},
			deleteFreqLock(idx) { // 删除 5G 锁频 FreqLock
				var vm = this;
				vm.freqLockList.splice(idx,1);
			},
			deleteCellLock(idx) { // 删除 5G 锁频 CellLock
				var vm = this;
				vm.cellLockList.splice(idx,1);
			},
			deleteBandLock(idx) { // 删除 5G 锁频 BandLock
				var vm = this;
				vm.bandLockList.splice(idx,1);
			},
			// cpe 设备型号改变事件
			cpeModelChange(){
				var vm = this;
				vm.queryParams.cpe_model = vm.cpeModel;
				vm.form.scanMode = '';
				vm.$refs.deviceListPairgrid.clear();
			},
			// 5GCPE  scan Mode为freq lock时  rat改变事件
			freqLockRatChange(){
				var vm = this;
				vm.freqForm.band = '1';
				this.$refs.freqForm.validateField('freq');
			},
			// 5GCPE  scan Mode为freq lock时  band改变事件
			freqLockBandChange(){
				var vm = this;
				this.$refs.freqForm.validateField('freq');
			},
		},
		mounted(){
			eventBus.$off('modify-task').$on('modify-task',this.init);
		}
	});
</script>