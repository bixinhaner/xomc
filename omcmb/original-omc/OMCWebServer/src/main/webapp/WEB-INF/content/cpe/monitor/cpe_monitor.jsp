<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<style>
	.newWindow{
		padding:0;
		height:56px;
		background:none;
	}
	.newWindow #winSettingProCpe{
		border:none;
		background:#dfeff8;		
	}
	.newWindow .window-shadow{
		background:none;
	}
	#winSettingProCpe>div{
		padding-left:26px;
		line-height:56px;
		color:#5892a0;
		font-weight:500;
	}
	#winSettingProCpe>div img{
		vertical-align:middle;
	}
	.cpe1ExportConfigItem{
		display: inline-block;
		width: 150px;
		line-height: 26px;
		vertical-align: middle;
	}
	.cpePanleSouth{
		height:30px;
		position:absolute;
		line-height:30px;
		bottom:0;
		right:0;
		left:0;
		background:#F4FAFD;
	}
	.cpePanleSouth > div{
		float:right;
		margin-right:20px;
	}
	#cpeSetting{/*pf*/
		background:#FFFFFF;
		position:absolute;
		top:40px;
		right:-1700px;
		width:930px;
		height:87%; 
		overflow-y:auto;
		z-index:100;
	}
	.titleStyle{
		font-size:14px;
		color:#92C8DF;
		height:13%;
		line-height:60px;
		margin-left:6%;
	}
	.qosOptionDetailsStyle{
		height:20%;
	} 
	.qosOptionDetailsLeft{
		display:inline-block;
		width:300px;
		margin-left:6%;
		margin-top:15px;
	}
	.qosOptionDetailsRight{
		display:inline-block;
		width:300px;
		margin-left:12%;
		margin-top:15px;
	}
	.inputTittleCss{
		font-size:14px;
		color:#797979;
		display: block;
		margin-bottom:2px;
	}
	.inputTipCss{
		font-size:12px;
		color:#D2D2D2;
		width:330px;
	}
	.inputDivCss{
		width:300px;
		display:block;
		height:26px;
	}
	.omcPageTitleContainer{
		height:40px;
		border-bottom:2px solid #F4FAFD;
		padding-left:20px;
		box-sizing:border-box;
		padding-top:13px;
	}
	.omcPageTitleContainer li{
		float:left;
		padding:0 20px;
		text-align:center;
		display:inline-block;
		height:26px;
		line-height:16px;
		font-size:14px;
		color:#92C8DF;
		margin-right:20px;
		cursor:pointer;
	}
	.omcPageTitleContainer li.default{
		border-bottom:2px solid #8DCAF9;
		box-sizing:border-box;
		color:#85A8BF;
		cursor:default;
	}
	.omcPageTitleContainer li.active{
		border-bottom:2px solid #92C8DF;
		box-sizing:border-box;
		color:#92C8DF;
	}
	.cpeApnTable_col{
		float:left;
		padding:10px 10px;
	}
	.title_row{
		height:50px;
		color:#85A8BF;
	}
	.apnNum_row{
		height:80px;	
		box-sizing:border-box;
	}
	.enable_col .apnNum_row,.defaultRouter_col .apnNum_row{
		padding-top:6px;
	} 
	.apnNum_col .apnNum_row{
		padding-top:4px;
	}
	.apnType_col select:disabled{
		background:#EAF1F4;
	}
	.showHideItem{
		position:absolute;
		width:700px;
		background:white;
		z-index:888;
		padding-top: 10px;
		padding-left: 10px;
		top: 100px;
		left: 0px;
		display:none;
		box-shadow:5px 10px 23px 0px rgba(201,212,231,0.50);
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
	.sortul{
		width:364px;
		height:45vh;
		padding:0px 0px 0px 0px;
		overflow:auto;
		overflow-x:hidden;
	}
	.sortul .cpe1ExportConfigItem{
		width:334px;
		height:30px;
		margin-top:2px;
		padding-left:30px;
		
		line-height:30px;
	}
	.sortul .cpe1ExportConfigItem span{
		display:block;
		width:20px;
		height:20px;
		float:right;
		margin-right:20px;
		margin-top:5px;
	}
	.sortul .cpe1ExportConfigItem span.movesTop:hover{
		cursor:pointer;
	}
	.handleMouseDown{
		 box-shadow:2px 2px 0 0 #C6C6C6;
	}
	.formBtnContainer{
		width:364px;
		height:58px;
		background:red;
		padding-top:22px;
	}
	.buttonGroup{
		width:178px;
		height:22px;
		margin: 0 auto;
		background:#CBF3FB;
		padding:7px 10px;
	}
	.buttonGroup div{
		width:84px;
		height:22px;
		line-height:22px;
		text-align:center;
		float:left;
	}
	.rightHideDiv{
		position:absolute;
		width:900px;
		background:#FFFFFF;
		box-shadow:2px 3px 16px rgba(158,200,222,0.5);
		right:-2000px;	
		top:0px;
		bottom:0px;
		z-index:100;
	}
	.cpeInformation{
		width:844px;
		position:absolute;
		right:-1000px;
		background:#FFFFFF;
		z-index:100;
		top:0px;
		bottom:0px;
	}
	.cpeSettingPanel,#apnSetting{
		width:930px;
		position:absolute;
		right:-1000px;
		background:#FFFFFF;
		z-index:100;
		top:0px;
		bottom:0px;
	}
	.historyGraph{
		display:inline-block;
		width:100%;
		height:28px;
		padding-left:0px;
	}
	#cpeQueryDiv ul.inputslist li{
		float: left;
	    height: 35px;
	    margin-right:80px;
	    margin-bottom:30px;
	    margin-top:10px;
	}
	#cpeQueryDiv ul.inputslist li label{
	    margin:0px 8px 0px 0px;   
	}
	.bottom-slider {
		position: absolute;
		bottom: -110px;
		width: 100%;
		padding: 30px 15px;
		background: #fff;
		box-shadow: 10px 0 50px rgba(0,0,0,0.15);
		transition: bottom .5s ease;
		z-index: 98;
	}
	.bottom-slider.show {
		bottom: 0px;
	}
	.selected-devices-info {
		padding: 15px 15px 15px 0;
		height: 400px;
		width: 360px;
		position: absolute;
		bottom: -450px;
		background: #fff;
		box-shadow: 10px 0 50px rgba(158,200,222,0.45);
		transition: bottom .5s ease;
		z-index: 90;
	}
	.selected-devices-info.show {
		bottom: 100px;
	}
	
	.enb-list-ctn {
		position:absolute;
		width: 100%;
		height: 100%;
		z-index: 100;
		top: 0;
		background-color: #fff;
		display: flex;
		flex-direction: column;
	}
	.list-title {
		font-size: 16px;
		color: #5A7B92;
		padding: 10px 0 10px 25px;
	}
	.list-body {
		width: calc(100% - 45px);
		padding-left: 25px;
		overflow: auto;
	}
	.list-body-title {
		background-color: #FBFBFB;
		color:#5A7B92;
		font-size: 12px;
		font-weight: bold;
		padding: 5px;
		border-bottom: 1px dashed silver;
	}
	.list-item-info {
		position: relative;
		padding: 6px 10px;
		border-bottom: 1px dashed silver;
	}
	.list-item-op {
		padding: 5px;
		display: inline-block;
		position: absolute;
		top: 0px;
		right: 5px;
		color: red;
		cursor: pointer
	}
	.changepwd-panel .prev {
		display: flex; 
		flex-direction: column;
		height: 100%;
	}
	.changepwd-panel .next {
		display: none; 
		flex-direction: column;
		height: 100%;
	}
	.changepwd-panel.modified .next {
		display: flex;
	}
	.changepwd-panel.modified .prev {
		display: none;
	}
	#pciLockDiv label{
		display:block;
		font-size:14px;
		font-weight:400;
		color:#797979;
		margin-bottom:5px;
	}
	.frequencyItem p{
		width:350px;
	}
	.msg_error{
		 color:red;
		 display:none;
	}
	.border_error{
		border:1px solid red;
	}
	.frequencyItem input{
		width:300px;
		height:26px;
		margin-bottom:3px;
	}
	.pciLockItem input{
		width:200px;
		height:26px;
		margin-bottom:3px;
	}
	.switch {
		background-color: #66CC66;
		margin-left:0px;
		float:none;
	}
	.statusContent{
		height:30px;
		border:1px solid #E9E9E9;
		border-radius:4px;
		padding-top:2px;
		display:inline-block;
		position:relative;
		float:left;
	}
	.statusContent .cmenu{
		width:auto;
	}
	.statusContent .cmenu-item{
		height:auto;
		line-height:normal;
	}
	.statusContent .cmenu-item span{
		margin-left:0px;
	}
	.statusType{
		padding:0 10px;
		height:28px;
		display:inline-block;
		float:left;
		margin-left:2px;
		line-height:28px;
	}
	.statusType span{
		font-size:12px;
		font-weight:bold;
	}
	.statusNum{
		height:28px;
		line-height:28px;
		display:inline-block;
		font-size:14px;
		color:#363B4E;
		font-weight:bold;
		float:right;
		margin-right:10px;
	}
	.greenType{
		background:#E9F8F1;
	}
	.greenType span{
		color:#38C846;
	}
	.redType{
		background:#FAEEEE;
	}
	.redType span{
		color:#E88282;
	}
	.el-icon-down:before{
		color:#999;
		cursor:pointer;
	}
	.highQueryArrow span{
		vertical-align:super;
	}
	.tabsTitle{
		border:none;
		color:#333;
	}
	#cpeInfo .el-icon-operation-lock:before{
		color:#F2B354;
	}
	.commonAddDevice .el-card__body{
		background:#FFFFFF;
		border:none;
		padding:20px 30px 0;
	}
	.commonAddDevice .el-form-item:first-child{
		margin-bottom:10px;
	}
	.commonAddDevice .el-form-item .el-form-item__label{
		line-height:26px;
	}
	.commonAddDevice .tipText{
		padding-bottom:26px;
		padding-top:14px;
		display:flex;
	}
	.infoTip:before{
		color:#CFCFCF;
		font-size:14px;
		margin-right:6px;
	}
	.commonAddDevice .el-card__footer{
		padding-left:30px;
		border:none;
	}
	#importDeviceCard .el-card__body{
		padding:30px 30px 0;
		background:#FFF;
		border:none;
		height:186px;
	}
	#importDeviceCard .el-card__body .uploadBox .el-input__suffix{
		top:4px;
	}
	#importDeviceCard .el-card__body .el-upload__tip{
		margin-top:0;
	}
	#importDeviceCard .el-card__body .uploadFormat{
		margin-left:10px;
		font-size:12px;
		color:#999999;
	}
	#importDeviceCard .el-card__header{
		padding:0 20px;
		border-bottom:1px solid #E9E9E9;
		color:#333;
	}
	#importDeviceCard .el-card__footer{
		border:none;
		height:24px;
		line-height:24px;
		padding:0 30px 20px;
	}
	#importDeviceCard .el-card__footer{
		border-top:none;
	}
</style>
<style>
	.list-left-area {
		width: calc(100% - 205px);
		border: 1px solid #E9E9E9;
	}
	.cpe-monitor-table {
		width: calc(100% - 205px);
	}
	.minisize .cpe-monitor-table, .minisize.list-left-area {
		width: 100%;
	}
	.list-left-area.expanded {
		width: 100%;
		border: none;
	}
	.fix-right-slide {
		flex-wrap: wrap;
		align-items: stretch;
		top: 36px;
		right: 0px;
		bottom: 0px;
		position: fixed;
		min-width: 200px;
		max-width: 200px;
		z-index: 100;
		background: #fff;
		border: 1px solid #E9E9E9;
	}
	.minisize .fix-right-slide, .expanded .slide-arrow.right, .minisize .slide-arrow.left {
		display: none;
	}
	.expanded .fix-right-slide {
		display: flex;
		position: absolute;
		max-width: 100%;
		top: 0px;
		left: 0px;
		border: 0px solid #F6F7FB;
		background: #F6F7FB;
	}
	.fix-right-slide > div {
		display: flex;
		flex-direction: column;
		flex: 1 auto;
		min-width: 200px;
		height: 200px;
		align-items: stretch;
		max-height: 200px;
	}
	.fix-right-slide .height-240 { 
		height: 240px; max-height: 240px;
	}
	.fix-right-slide > div:not(:nth-child(3)) {
		border-bottom: 1px solid #e9e9e9;
	}
	.expanded .fix-right-slide > div {
		height: calc(50% - 10px);
		max-height: 49%;
		border: 1px solid #E9E9E9;
		margin: 5px;
		background: #fff;
		border-radius: 5px;
	}
	.slide-arrow {
		padding: 3px 0 3px 8px;
		position: fixed;
		min-width: 20px !important;
		max-width: 20px !important;
		max-height: 18px !important;
		top: calc(50% + 10px);
		right: 201px;
		z-index: 100;
		background-color: #4D84FF !important;
		border-radius: 15px 0 0 15px;
		box-shadow: 2px 0 15px rgba(0,0,0,.2);
	}
	.slide-arrow.right {
		right: 172px;
		transform: rotate(180deg);
	}
	.minisize .slide-arrow {
		right: 10px;
	}
	.minisize .slide-arrow {
		transform: rotate(0deg);
	}
	.expanded .slide-arrow {
		right: 15px;
		border-radius: 15px 0 0 15px !important;
	}
	.slide-arrow:hover::before {
		position: absolute;
		display: inline-block;
		min-width: 80px;
		content: '<%=rb.getString("ZhanKai")%>';
		bottom: -25px;
		left: -5px;
		font-size: 14px;
		color: #4d84ff;
	}
	.slide-arrow.right:hover::before {
		position: absolute;
		content: '<%=rb.getString("GuanBi")%>';
		left: -60px;
		transform: rotate(180deg);
	}
	.expanded .slide-arrow:hover::before {
		content: '<%=rb.getString("GuanBi")%>';
	}
	.minisize .slide-arrow.right:hover::before {
		left: -5px;
		min-width: 40px;
		content: '<%=rb.getString("ZhanKai")%>';
		transform: rotate(0deg);
	}
	.slide-arrow i {
		font-size: 12px;
		transform: rotate(90deg);
	}
	.expanded .slide-arrow i {
		transform: rotate(-90deg);
	}
	.slide-arrow i::before {
		color: #fff !important;
	}

	.fix-right-slide .no-data::before {
		content: '<%=rb.getString("MeiShuJu")%>';
		position: absolute;
		top: 50%;
		left: calc(50% - 20px);
		z-index: 100;
	}
	
	.showHideItem {
		padding-top: 10px;
		padding-left: 10px;
		width: 700px;
	}
	.col-group {
		margin: 5px 10px;
		display: flex;
		flex-wrap: wrap;
	}
	.col-group .el-checkbox {
		min-width: 140px;
	}
	.col-group .el-checkbox__label {
		font-size: 12px;
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
		font-weight: bold;
		margin-left: 10px;
	}
	.showHideItem .el-icon-close1::before {
		color: #333;
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
	.messager-button a:nth-child(2):hover{
		background:#F2F9FF;
		color:#4D84FF;
		border:1px solid #2A61DB
	}
	.messager-button a:nth-child(2){
		background:#FFFFFF;
		color:#666;
		border:1px solid #DCDFE6
	}
	.pciClass{
		display: inline-block;
		height: 18px;
		width: 18px;
		text-align: center;
		line-height: 18px;
		border: 1px solid #DCDFE6;
		border-radius: 2px;
	}
	.pciClass .el-icon::before{
		font-size: 16px;
	}
	.pciClass:hover{
		border: 1px solid #4D84FF;
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

	td[field=module_version], td[field=SOFTWARE_VERSION] {
		position: relative;
	}
	.version-details {
		display: none;
		position: absolute;
		right: 10px;
		padding: 20px 20px 10px 15px;
		box-shadow: 2px 2px 5px silver;
		background: #fff;
		border-radius: 3px;
		z-index: 100;
	}
	.version-details.normal {
		display: block;
		position: relative;
		box-shadow: none;
		padding: 0 5px;
	}
	.version-details .item {
		display: block;
		width: auto;
		min-width: 100px;
		padding: 2px 0;
		border-bottom: 1px dashed silver;
	}
	.version-details .panel_close {
		margin-top: -15px;
	}
	.enbMonitorForm{
		margin-left:10px;
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
		text-overflow:ellipsis;
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

	.active-top {
		background: #fff;
		width: calc(100% - 205px);
	}
	.active-top > span {
		padding: 0 10px;
	}
	.active-top > span.active {
		border-top: 2px solid #4D84FF;
		border-bottom: none;
	}

	.hide-fixed .fix-right-slide,
	.hide-fixed .slide-arrow.left,
	.hide-fixed .slide-arrow.right,
	.hide-fixed #addOrImport,
	.hide-fixed #cpe_export_op {
		display: none;
	}
	.hide-fixed.list-left-area{
		width: 100%;
	}
</style>
<%-- CPE监控页面 --%>
<div class="panelDefault list-left-area" id="cpeInfo" style="overflow:hidden;">
	<div id="addOrImport" style="position:relative">
		<!-- 右上角新建按钮 -->
		<div class="circleIcon CODE_CPE_DEVICE hidden" style="right:100px;top:10px;">
			<span class="el-icon el-icon-circle-add" @click="addDevice"></span>
			<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
		</div>
		<!-- 右上角导入按钮 -->
		<div class="circleIcon CODE_CPE_DEVICE hidden" style="right:50px;top:10px;">
			<span class="el-icon el-icon-circle-import" @click="importDevice"></span>
			<div class="titleButtonText"><%=rb.getString("DaoRu")%></div>
		</div>
		
		<!-- 新建 设备-->
		<div>
			<transition name='el-zoom-in-top'>
				<el-card v-show='showAddDeviceCard' class="commonAddDevice" id='addDeviceCard' style='width:600px;height:auto;z-index:999;box-shadow:0 0 10px rgba(0, 0, 0, 0.16);position:absolute;right:20px;top:50px;'>
					<div slot='header'>
						<span><%=rb.getString("TianJiaCPE")%></span>
						<span class='el-icon el-icon-close' style='float:right;font-size:16px;' @click='closeAddDevice'></span>
					</div>
					<div>
						<el-form  :model='enodebForm' :rules="enodebRules" ref="enodebForm" label-position="left">							
							<el-form-item label='MAC' prop='serialnumber'>
								<el-input type="textarea" v-model="enodebForm.serialnumber"></el-input>
							</el-form-item>
							<div class="tipText">
								<span class="el-icon el-icon-circle-info infoTip"></span>
								<span><%=rb.getString("cpeZhuCeTiShiWenZi")%></span>
							</div>
							<el-form-item label='<%=rb.getString("SheBeiZuMingCheng")%>' prop='groupId' label-width="120px">
								<el-select v-model="enodebForm.groupId" >
									<el-option v-for="item in deviceGroupSelections" :key="item.id" :label="item.group_name" :value="item.id"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item label='<%=rb.getString("LianJieTiaoJian")%>' label-width="120px">
								<el-select v-model="enodebForm.link" >
									<el-option v-for="item in deviceLinkOptions" :key="item.value" :label="item.text" :value="item.value"></el-option>
								</el-select>
							</el-form-item>
						</el-form>
						
					</div>
					<div slot='footer'>
						<el-button-group size="mini">
			    			<el-button type="primary" size="mini" @click="addeNodeb"><%=rb.getString("QueDing")%></el-button>
			    			<el-button size="mini" @click="closeAddDevice"><%=rb.getString("QuXiao")%></el-button>
			    		</el-button-group>
					</div>					
				</el-card>
			</transition>
		</div>
		
		<!-- 导入文件框 -->
		<div>
			<transition name='el-zoom-in-top'>
				<el-card v-show='showImportCard' class="" id='importDeviceCard' style='width:600px;height:300px;border:1px solid #DCDFE6;box-shadow:0 0 10px rgba(0, 0, 0, 0.16);position:absolute;right:20px;top:50px;z-index:999;'>
					<div slot='header'>
						<span><%=rb.getString("DaoRuSheBei")%></span>
						<span class='el-icon el-icon-close' style='float:right;font-size:16px;' @click='closeImportDevice'></span>
					</div>
					<div>
						<div style="display:flex;" class="uploadBox">
							<label style='display:inline-block;margin-right:30px;'><%=rb.getString("WenJian")%></label>
							<el-upload :on-success='checkFile' :on-change="fileChange" :show-file-list=false ref="upload"
							     :action="uploadFileURL" :data="fileParams" name="uploadFile" :auto-upload="false">
								<el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' style="width:260px;">
									<a slot="suffix" class="el-icon el-icon-operation-import importBox" @click="fileSelect"></a>
								</el-input>
								<span class="uploadFormat">.xlsx /.csv</span>
								<div slot="tip" class="el-upload__tip" v-show="!typeFlag"><%=rb.getString("ZhiZhiChiXLSXCSVWenJian")%></div>
								<div slot="tip" class="el-upload__tip" v-show="selectFlag"><%=rb.getString("QingXianXuanZeWenJian")%></div>
								<a slot="trigger" ref="file_up"></a>
							</el-upload>					
						</div>
						
						<div style='color:#999;line-height:24px;padding:6px 0 0 0px;display:flex;'>
							<div>
								<span class='el-icon el-icon-circle-info infoTip' style='font-size:14px;'></span>
								<span><%=rb.getString("DaoRuWenJianTiShi")%></span>
							</div>
							
						</div>
						<span style="cursor:pointer;" @click="exportTemplate">
							<span style='vertical-align:top' class='el-icon el-icon-common-download'></span>
							<span style='color:#363B4E;text-decoration:underline'><%=rb.getString("DaoChuMuBan")%></span>
						</span>
						<div style="padding-top:20px;">
							<label style='display:inline-block;margin-right:30px;'><%=rb.getString("SheBeiZuMingCheng")%></label>
							<el-select v-model="importGroupId">
								<el-option v-for="item in deviceGroupSelections" :key="item.id" :label="item.group_name" :value="item.id"></el-option>
							</el-select>
						</div>
					</div>
					<div slot='footer'>
						<el-button-group size="mini">
			    			<el-button type="primary" size="mini" @click="uploadDevice"><%=rb.getString("QueDing")%></el-button>
			    			<el-button size="mini" @click="closeImportDevice"><%=rb.getString("QuXiao")%></el-button>
			    		</el-button-group>
					</div>
					
				</el-card>
			</transition>
		</div>			
	</div>

	<!-- 右上角导出按钮 -->
	<div class="circleIcon placeholder-bt" placeholder="<%=rb.getString("DaoChu")%>" style="top: 10px;" id="cpe_export_op" @click="showExport">
		<el-popover trigger="click" placement="bottom-end">
			<div class="el-card__header">
				<%=rb.getString("DaoChu")%>
				<span style="color: 999;font-weight: normal;margin-left: 5px;">(<%=rb.getString("SuoYouCanShu")%>)</span>
				<span style="font-size: 14px;" class="el-icon el-icon-close" onclick="document.body.click();"></span>
			</div>
			<div class="export-content" style="width: 800px;"></div>
			<span slot="reference" class="el-icon-circle-export el-icon"></span>
		</el-popover>
	</div>

	<div class="tabsTitle active-top" id="eNb_tabs_area">
		<span tabtit="cpemonitor" onclick="turnTabs(this);showFixed();" class="active"><%=rb.getString("LieBiao")%></span>
		<span tabtit="cpetopo" onclick="turnTabs(this);loadCpeTopo();hideFixed();"><%=rb.getString("Topo")%></span>
	</div>
	<div class="tabsContentDiv" style="top: 30px;border: none;">
		<div class="cpemonitor" style="display: block;">
			<table id="tableHomeCpeList" class="cpe-monitor-table"></table>
		</div>

		<div class="cpetopo nocontent-loading" id="cpe_topo_ctner"></div>
	</div>


	<div class="fix-right-slide">
		<div style="width: 100%;position: relative;">
			<div id="cpe_online_count_rate" style="position: absolute;left: 120px;z-index: 100;font-size: 12px;top: 13px;"></div>
			<div id="online_status_chart" style="width: 100%;height: 100%;"></div>
		</div>
		<div style="width: 45%;" class="height-240">
			<div id="module_name_chart" style="width: 100%;height: 100%;"></div>
		</div>
		<div style="width: 45%;">
			<div id="device_time_chart" style="width: 100%;height: 100%;"></div>
		</div>
	</div>
	<div class="slide-arrow left" onclick="toggleExpand()">
		<i class="el-icon el-icon-down"></i>
	</div>
	<div class="slide-arrow right" onclick="minisize()">
		<i class="el-icon el-icon-down"></i>
	</div>
	
	<%-- 右键 - 设置 弹出框  --%>
	<div id="cpeSetting"  class="slidebarPanel"></div>
    <div id="winSettingProCpe" style="width:100%;height:56px;position:absolute;background:#dfeff8;bottom:0px;display:none">
		<div class="setting">
			<img src="${ctx}/images/set-loading.gif"/>
			<span style="margin-left:20px;"><%=rb.getString("CpeZhengZaiSheZhi")%></span> 
		</div>
	    <div class="set" style="display:none;">
	    	<img src="${ctx}/images/loading-finish.png"/>
	    	<span style="margin-left:20px;">CPE setup completed!</span> 
	    </div>   
	</div> 
	<!-- CPE历史数据图表 -->
	<div id="CPEHistoryCharts" class="rightHideDiv slidebarPanel" style="right: -1300px;"></div>
	<div id='cpeInformation' class='cpeInformation slidebarPanel'></div>
	
	<!-- cpe设置 -->
	<div id='cpeSettingOption' class='cpeSettingPanel slidebarPanel'></div>
	<!-- apn设置 -->
	<div id='apnSetting' class='slidebarPanel'></div>
	<%-- 批量密码设置 --%>
	<div class="bottom-slider">
		<div style="display: flex;">
			<div style="flex: auto;padding: 10px;display: flex;">
				<%=rb.getString("YiXuanSheBei")%>（<span id="device_count" style="color: blue;"></span>）
				<span id="sliderArrow" class="el-icon el-icon-circle-down" style="font-size:16px; cursor: pointer;" onclick="showDeviceList()"> </span>
			</div>
			<div>
				<div class="linkbuttonGroup">
					<%-- <a class="linkbutton linkbutton_trend CODE_CPE_SYNCHRONIZE hidden" onclick="cpeRefreshBatch()">
						<span><%=rb.getString("TongBu")%></span>
					</a> --%>
					<a class="linkbutton linkbutton_trend CODE_CPE_REBOOT hidden" onclick="cpeRebootBatch()">
						<span><%=rb.getString("ChongQi")%></span>
					</a>
					<a class="linkbutton linkbutton_trend CODE_CPE_CHANGE_PASSWORD hidden" onclick="toModifyPassword()">
						<span><%=rb.getString("XiuGaiMiMa")%></span>
					</a>
					
					
					<a id="lte_turbo_enable" href="#" class="linkbutton" onclick="ltmkai('1')" style="margin-right:-4px;"><span>LTE-TURBO  <%=rb.getString("QiYong")%></span></a>
			    	<a id="lte_turbo_disable" href="#" class="linkbutton linkbutton_nowanna" onclick="ltmkai('0')" ><span>LTE-TURBO <%=rb.getString("JinYong")%></span></a>
					<a class="linkbutton linkbutton_nowanna" onclick="delAllSelectedRecord()" style='margin-right:35px;'>
						<span><%=rb.getString("QuXiao")%></span>
					</a>
				</div>
			</div>
		</div>
	</div>
	
	<div class="selected-devices-info">
		<div class="enb-list-ctn">
			<div class="list-title">
				<%=rb.getString("YiXuanSheBei")%>
				<span style="float: right;padding-right: 15px;" onclick="showDeviceList(false)"><i class="el-icon el-icon-close"></i></span>
			</div>
			<div style="width: calc(100% - 44px);padding-left: 25px;">
				<div class="list-body-title">
					<%=rb.getString("CPEName")%> + <%=rb.getString("CPEXuLieHao")%>
					<div class="el-icon el-icon-operation-delete" title="Delete All" style="font-size:14px;margin-left:5px;float: right;padding-left: 50px;" onclick="delAllSelectedRecord()"><%=rb.getString("QingKong")%></div>
				</div>
			</div>
			<div class="list-body"></div>
		</div>
	</div>

	<div id="modify_password_dialog" class="changepwd-panel" title="Confirm">
		<div class="prev">
			<div style="padding: 20px;">
				<%=rb.getString("QueRenXiuGaiMiMa")%>
			</div>
			<div style="flex: auto;padding: 20px;position: relative;">
				<span style="margin-right:20px;font-weight: bold;"><%=rb.getString("XinMiMa")%></span>
				<input id="pwd_input" style="width: 200px; height: 26px;border:1px solid #E9E9E9" type="password" autocomplete="new-password" onblur="validatePasswordFun()"/>
				<span id="pwd_type_change" class="el-icon el-icon-operation-hide" style="display:inline-block;width:20px;height:20px;vertical-align:middle;" ></span>
				<span id="pwd_error" style="position: absolute; left: 100px;bottom: 0px;color: red;"></span>
			</div>
			<div style="text-align: right;">
				<div class="bt-group linkbuttonGroup" style="margin: 15px 20px;">
					<a class="linkbutton" onclick="changepwdFun()"><span><%=rb.getString("QueDing")%></span></a>
					<a class="linkbutton linkbutton_nowanna" onclick="cancelModify()"><span><%=rb.getString("QuXiao")%></span></a>
				</div>
			</div>
		</div>
		
		<div class="next">
			<div style="padding: 20px;">
				<%=rb.getString("RenWuYiJianLi")%><span id="created_task_name"></span>
			</div>
			<div style="flex: auto;padding:10px 20px;">
				<%=rb.getString("XiuGaiMiMaChengGongTiShi")%>
			</div>
			<div style="text-align: right;">
				<div class="bt-group linkbuttonGroup" style="margin: 15px 20px;">
					<a class="linkbutton" onclick="toTaskPage()"><span><%=rb.getString("QianWang")%></span></a>
					<a class="linkbutton linkbutton_nowanna" onclick="cancelModify()"><span><%=rb.getString("QuXiao")%></span></a>
				</div>
			</div>
		</div>
	</div>
</div>

<!-- CPE监控工具栏	 -->
<div id="toolbar_tableHomeCpeList" class="toolbarContainer" style="position:relative">
	<span class="el-icon el-icon-operation-settings" style="font-size:20px;position:absolute;left:5px;z-index:89;width:30px;height: 40px;" :style='{top:setTop}' onclick="ColomnStatus('showHideItemCpe')"></span>
	<!-- 显示隐藏列 -->
	<div class="showHideItem" id="showHideItemCpe">
		
		<div class="flex-ctn" style="padding-left: 10px;">
			<div class="select-all-cls" style="padding-left: 10px;">
				<el-checkbox :indeterminate="!colAll" v-model="colAll" @change="colAllChange"></el-checkbox> 
				<span><%=rb.getString("QuanXuan")%></span>
			</div>
			<div>
				<div class="select-all-cls">
					<i :class="{'el-icon':true,'el-icon-close1':expanded.device,'el-icon-open':!expanded.device}" @click="expanded.device = !expanded.device"></i>
					<el-checkbox :indeterminate="form.device.length<deviceCol.length" v-model="deviceAll" @change="deviceAllChange"></el-checkbox> 
					<span><%=rb.getString("SheBeiXinXi")%></span>
				</div>
				<el-checkbox-group v-show="expanded.device" class="col-group" v-model="form.device">
					<el-checkbox v-for="item in deviceCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
				</el-checkbox-group>
			</div>
			<div>
				<div class="select-all-cls">
					<i :class="{'el-icon':true,'el-icon-close1':expanded.lte,'el-icon-open':!expanded.lte}" @click="expanded.lte = !expanded.lte"></i>
					<el-checkbox :indeterminate="form.lte.length<lteCol.length" v-model="lteAll" @change="lteAllChange"></el-checkbox> 
					<span><%=rb.getString("LTEZhuangTai")%></span>
				</div>
				<el-checkbox-group v-show="expanded.lte" class="col-group" v-model="form.lte">
					<el-checkbox v-for="item in lteCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
				</el-checkbox-group>
			</div>
			<div>
				<div class="select-all-cls">
					<i :class="{'el-icon':true,'el-icon-close1':expanded.lan,'el-icon-open':!expanded.lan}" @click="expanded.lan = !expanded.lan"></i>
					<el-checkbox :indeterminate="form.lan.length<lanCol.length" v-model="lanAll" @change="lanAllChange"></el-checkbox> 
					<span><%=rb.getString("LANZhuangTai")%></span>
				</div>
				<el-checkbox-group v-show="expanded.lan" class="col-group" v-model="form.lan">
					<el-checkbox v-for="item in lanCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
				</el-checkbox-group>
			</div>
			<div>
				<div class="select-all-cls">
					<i :class="{'el-icon':true,'el-icon-close1':expanded.location,'el-icon-open':!expanded.location}" @click="expanded.location = !expanded.location"></i>
					<el-checkbox :indeterminate="form.location.length<locationCol.length && form.location.length>0" v-model="locationAll" @change="locationAllChange"></el-checkbox> 
					<span><%=rb.getString("WeiZhi")%></span>
				</div>
				<el-checkbox-group v-show="expanded.location" class="col-group" v-model="form.location">
					<el-checkbox v-for="item in locationCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}">{{item.label}}</el-checkbox>
				</el-checkbox-group>
			</div>
		</div>
		<div class="windowButtonGroup" style="float:none !important;margin:20px 0 20px 20px">
			<a class="linkbutton linkbutton_trend" @click="columnConfig"><span><%=rb.getString("QueDing")%></span></a>
			<a class="linkbutton linkbutton_nowanna" @click="closeConfig"><span><%=rb.getString("QuXiao")%></span></a>
		</div>
	</div>
	<!-- 高级查询 -->
	<el-form ref="cpeMonitorForm" :model="cpeMonitorForm" class='enbMonitorForm' label-position="left" label-width="110px">
		<div>
			<div class='selectItem' style='display:inline-block'>
				<el-select v-model="queryValue" @change='changeQuery'>
					<el-option v-for="item in queryOptions" :key="item.value" :label="item.label" :value="item.value"></el-option>
				</el-select>
			</div>
			<div style='margin-left:-3px;vertical-align:bottom;display:inline-block'>
				<div class='queryGroup'>
					<el-input v-model="search_text" @keyup.enter.native="query" class='pairgrid-query' :placeholder='queryTip'></el-input>
					<i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
				</div>
			</div>
		</div>
		<div style='margin-top:10px;background:#F6F7FB;padding:6px 0px 0px 0px;display:flex;border:1px solid #E9E9E9'>
			<div class='selectContent'>
				<el-collapse-transition>
					<div :style="{height:divHeight}" style="overflow:hidden">
						<el-form-item label="<%=rb.getString("LianJieZhuangTai")%>" prop="connection_status">
							<el-select v-model="cpeMonitorForm.connection_status" @change='changeStatus("connection_status")' multiple collapse-tags>
								<el-option v-for="item in onlineOptions" :key="item.value" :label="item.label" :value="item.value"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label="<%=rb.getString("SoftwareVersion")%>" label-width="110px" prop="software_version">
							<el-select v-model="cpeMonitorForm.software_version" @change='changeStatus("software_version")' multiple collapse-tags>
								<el-option v-for="item in softwareOptions" :key="item.value" :label="item.text" :value="item.value"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label="<%=rb.getString("CPELeiXing")%>" label-width="110px" prop="cpeMonitorModule">
							<el-select v-model="cpeMonitorForm.cpeMonitorModule" @change='changeStatus("cpeMonitorModule")'>
								<el-option v-for="item in moduleOptions" :key="item.value" :label="item.text" :value="item.value"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label="<%=rb.getString("ChanPinXingHao")%>" label-width="110px" prop="product_model">
							<el-select v-model="cpeMonitorForm.product_model" @change='changeStatus("product_model")' multiple collapse-tags>
								<el-option v-for="item in productModelOptions" :key="item.value" :label="item.text" :value="item.value"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label="<%=rb.getString("SheBeiZu") %>" prop="group_id">
							<el-select v-model="cpeMonitorForm.group_id" @change='changeStatus("group_id")' multiple collapse-tags>
								<el-option v-for="item in deviceGroupOptions" :key="item.value" :label="item.text" :value="item.value"></el-option>
							</el-select>
						</el-form-item>
					</div>
				</el-collapse-transition>
			</div>
			<el-button-group size="mini" style='margin-right:10px;'>
				<span v-show="clientWidth" @click="slidedown" class='el-icon' :class="slideIcon" style='font-size:18px;float:left;margin-right:10px;margin-top:3px;'></span>
				<el-button size="mini" @click="resetCPEQuery"><%=rb.getString("ChaXunChongZhi")%></el-button>
   			</el-button-group>
		</div>
	</el-form>
	<%-- <div class="admin_query_head" style="display:inline-block;">
		<form id="cpequeryform">
			<div class="easyui-query" tips="<%=rb.getString("GaoJiChaXun") %>" 
				name="search_text" 
				inputId="cpeSearchText" targetId="cpeQueryDiv" 
				placeholder="<%=rb.getString("CPEName")%>/<%=rb.getString("HostName")%>/<%=rb.getString("CPEMacAddress")%>/IMSI/<%=rb.getString("CPEBianMa")%>" 
				data-options="query: function(){ queryCPETable('e'); }" style='display:inline-block;float:left;'></div>
				
			<div id="cpeQueryDiv" style="width:100%;padding:20px 20px 10px 40px;display:none;position:absolute;top:45px;left:0px;z-index:100;background:#FFFFFF;-webkit-box-shadow:0px 10px 32px rgba(158,200,222,0.35);">
				<ul class="inputslist">
					<li atype="SERIAL_NUMBER">
						<label><%=rb.getString("CPEBianMa")%>:</label><br>
						<input type="text" id="cpeSN" name="serial_number" class="border-box border" style="height:26px;width:200px;"/>
					</li>
					
					<li atype="CPE_NAME">
						<label><%=rb.getString("CPEName")%>:</label><br>
						<input type="text" id="cpeName" class="border-box border" name="CPE_NAME" style="height:26px;width:200px;"/>
					</li>
					<li atype="CELL_NAME">
						<label><%=rb.getString("HostName")%>:</label><br>
						<input type="text" id="cellName" class="border-box border" name="HOST_NAME" style="height:26px;width:200px;"/>
					</li>
					<li atype="IMSI">
						<label>IMSI:</label><br>
						<input type="text" id="cpeimsi" class="border-box border" name="IMSI" style="height:26px;width:200px;"/>
					</li>
					<li atype="IPADDRESS">
						<label><%=rb.getString("IPDiZhi")%>:</label><br>
						<input type="text" id="cpeipAddr" class="border-box border" name="IP_ADDRESS" style="height:26px;width:200px;"/>
					</li>
					<li atype="MACADDRESS">
						<label><%=rb.getString("MACDiZhi")%>:</label><br>
						<input type="text" id="cpemacaddr" class="border-box border" name="MAC_ADDRESS" style="height:26px;width:200px;"/>
					</li>
					<li  atype="CELL_IDENTITY">
						<label>ECI:</label><br>
						<input type="text" id="cpecellid" class="border-box border" name="CELL_IDENTITY" style="height:26px;width:200px;"/>
					</li>
					<li  atype="pci">
						<label>PCI:</label><br>
						<input type="text" id="pciInput" class="border-box border" name="pci" style="height:26px;width:200px;"/>
					</li>
					<li>
						<label><%=rb.getString("SoftwareVersion")%>:</label><br>
						<select id = "CPESoftVersion" class="easyui-combobox border border-box combobox-f combo-f textbox-f" data-options="editable:false" name="software_version" style="padding-top:0px;height:26px;width:200px;"></select>
					</li>
					<li>
						<label><%=rb.getString("SheBeiZu")%>:</label><br>
						<select id = "CPEDeviceGroup" class="easyui-combobox border border-box combobox-f combo-f textbox-f" data-options="editable:false" name="group_id" style="padding-top:0px;height:26px;width:200px;"></select>
					</li>
					<li>
						<label><%=rb.getString("LianJieZhuangTai")%>:</label><br>
						<select id = "CPEconnStatus" class="easyui-combobox border border-box combobox-f combo-f textbox-f" data-options="editable:false" name="connection_status" style="padding-top:0px;height:26px;width:200px;">
							<option value=""><%=rb.getString("QuanBu")%></option>
							<option value="1"><%=rb.getString("LianJieZhengChang")%></option>
							<option value="0"><%=rb.getString("LianJieDuanKai")%></option>
							<option value="3"><%=rb.getString("TongBuZhong")%></option>
							<option value="2"><%=rb.getString("TongBuShiBai")%></option>
						</select>
					</li>
					<li atype="LGWIP">
						<label>LGW IP:</label><br>
						<input type="text" id="cpeLGWIP" class="border-box border" name="lgw_ip" style="height:26px;width:200px;"/>
					</li>
					<li atype="LGWMAC">
						<label>LGW MAC:</label><br>
						<input type="text" id="cpeLGWMAC" class="border-box border" name="lgw_mac" style="height:26px;width:200px;"/>
					</li>
				</ul>
				<div class="linkbuttonGroup" style="margin-bottom:20px">
					<a href="#" class="linkbutton linkbutton_trend" onclick="queryCPETable()"><span><%=rb.getString("ChaXun")%></span></a>
					<a href="#" class="linkbutton linkbutton_nowanna" onclick="cpeResetQueryInput()"><span><%=rb.getString("ChaXunChongZhi")%></span></a>
				</div>
			</div>
			<div class='statusContent' style='width:200px;margin-left:65px;display: none;'>
				<div class='statusType greenType' style='width:75px;'>
					<span class='connectionName'><%=rb.getString("ShouYe_ZaiXian")%></span>
					<span class='el-icon-down el-icon' style='margin-left:10px;float:right;margin-right:5px;margin-top:8px;' onclick='showOnline()'></span>
				</div>
				<div class='statusNum' id='onlineStatusNumCpe'></div>
				<div class='cmenu item-left onlineMenu' style='position:absolute;left:0px;top:35px;display:none'>
					<div class='cmenu-item' style='padding:9px 35px 9px 10px !important' onclick="onlineClick('online')"><span><%=rb.getString("ShouYe_ZaiXian")%></span></div>
					<div class='cmenu-item' style='padding:9px 35px 9px 10px !important' onclick="onlineClick('offline')"><span><%=rb.getString("ShouYe_BuZaiXian")%></span></div>
				</div>
			</div>
		</form>
	</div> --%>

</div>

<%-- 右键菜单－CPE列表 --%>
<div id="rowMenuTableHomeCpeList" class="easyui-menu">
	<div onclick="goCpeDetailParamInfoWin('cpeInformation')"><%=rb.getString("XinXi")%></div>
	<div onclick="editCpe('cpeSetting')"><%=rb.getString("SheZhi")%></div>
	<div onclick="refreshCpe()"><%= rb.getString("Kai")%></div>
	<div onclick="refreshCpe()" id="tongbuCpe"><%=rb.getString("TongBu")%></div>
	<div class="menu-sep" id="firstline"></div>
	<div onclick="cpeFreqLock()" id="freqLock" data-options="disabled:false"><%=rb.getString("SuoPin")%></div>
	<div class="menu-sep"></div>
	<c:if test="${no_rebootCpe == 1}">
    	<div title="<%=rb.getString("MeiYouQuanXian")%>" data-options="disabled:true"><%=rb.getString("ChongQi")%></div>
    </c:if>
    <c:if test="${no_rebootCpe != 1}">
    	<div id='chongqiCpe' onclick="cpeReboot()"><%=rb.getString("ChongQi")%></div>
    </c:if>
</div>

	<!-- 菜单生成 -->
<div class="wrap">
    <div id="cpemn"></div>
</div>

<%-- 表单-用于导出CPE列表 --%>
<form id="formExportCpeList" style="display:none" method="post"
      action="${ctx}/cell/CPE/exportCpesToCSV.action">
</form>
<script>
var addOrImport = new Vue({
	el: '#addOrImport',
	data(){
		var validatorNum = (rule,value,callback) => {
			
			var serialNumber = (value||'').replace(/\s/g,''),
				lastIndex = serialNumber.lastIndexOf(';'),
				length = serialNumber.length;
			
			if(length - lastIndex == 1) {
				serialNumber = serialNumber.substring(0,lastIndex);
			}
			
			var serialNumberArr =  serialNumber.split(";");
			var temp = /^[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}$/;
			var noColTemp = /^([A-Fa-f0-9]{2}){6}$/;
		    if (serialNumber == null || serialNumber.length == 0) {
				callback(new Error('<%=rb.getString("QingShuRuZhengQueMac")%>'));
			}else {
				var nameFlag = serialNumberArr.every(function(item,index){
					return (temp.test(item) ||  noColTemp.test(item)) 
				})
				if(nameFlag){
					callback()
				}else{
					callback(new Error('<%=rb.getString("QingShuRuZhengQueMac")%>'));
				}
			}
		};
		return {
			showAddDeviceCard:false,
			enodebForm:{
				serialnumber:'',
				groupId:'',
				link:''
			},
			enodebRules:{
				serialnumber:[
					{validator:validatorNum,trigger:'blur'}
				]
			},
			type:'',
			deviceGroupSelections:[],
			deviceLinkOptions: [
				{text:'NLOS',value:'nlos'},
				{text:'PLOS',value:'plos'},
				{text:'LOS',value:'los'}
			],
			groupId:'',
			defaultGroupId:'',
			//导入文件
			importGroupId:'',
			showImportCard:false,
			selectFlag:false,        //标识是否选择了文件 
			typeFlag:true,           //校验已选择的文件格式 
			fileName:'',
			fileParams:{},            //上传文件时自定义的参数   
			uploadFileURL: '${ctx}/cell/CPE/uploadFile.action?importType=append',
		}
	},
	
	methods: {			
		addDeviceInit(){
			var vm = this;
			axios.post("${ctx}/system/deviceGroup/getDeviceGroupList.action").then((res) => {
				var data = res.data.rows;
				if(data){
					vm.deviceGroupSelections = data; 						
					vm.defaultGroupId = data[0].id;
					vm.enodebForm.groupId = vm.defaultGroupId;
					vm.importGroupId = vm.defaultGroupId;
				}					
			})				
		}, 
		
		addDevice(){
			var vm = this;
			vm.openOrCloseAdd();
			vm.showAddDeviceCard = true;
			vm.showImportCard = false;
		},			
		closeAddDevice(){
			var vm = this;
			vm.openOrCloseAdd();
			vm.showAddDeviceCard = false;
			
		},
		openOrCloseAdd(){
			var vm = this;
			vm.enodebForm.link = '';
			vm.$refs.enodebForm.resetFields();
			vm.enodebForm.groupId = vm.defaultGroupId;
		},

		addeNodeb(){
			var param={} , vm = this , url ,id;
			var message = "<%=rb.getString("TianJiaSheBeiChengGong")%>";
				id = vm.enodebForm.groupId;
				url = "${ctx}/cell/CPE/addAndAssignCpe.action",
				mac = vm.enodebForm.serialnumber,
				link_condition = vm.enodebForm.link;
			
			if(mac) {
				mac = mac.split(';').map(function(item){
					return (item.trim().toLocaleUpperCase().match(/[a-zA-Z0-9]{2}/g) || []).join(':');
				}).join(';');
			}				
			 vm.$refs.enodebForm.validate((valid) => {
				if(valid){
					axios.post(url,stringify({
						"group_id":id,
						"macAddress":mac,
						"link_condition":link_condition
					})).then(function(response){
						let data = response.data;
						if (data.success){
							//刷新表格,关闭新建设备弹窗
							$('#tableHomeCpeList').datagrid('reload');
							vm.closeAddDevice();
							vm.$message({
								message: message,
								type:'success',
							})
						}else {
							vm.$message.error(data.message)
						}
						
					}).catch(function(error){})
					
				}else{
				}
			}) 				
		},

		 
		// 打开导入弹出框
		importDevice(){
			var vm = this;
			vm.openOrCloseImport();
			this.showImportCard = true;
			this.showAddDeviceCard = false;
		},
		// 关闭导入弹出框
		closeImportDevice(){
			var vm = this;
			vm.openOrCloseImport();
			vm.showImportCard = false;
		},
		openOrCloseImport(){
			var vm = this;
			vm.typeFlag = true;
			vm.selectFlag = false;	
			vm.fileName = '';
			vm.fileParams.FileName = '';
			vm.importGroupId = vm.defaultGroupId;
			vm.fileParams.group_id = vm.defaultGroupId; 
		},
		
		/**
		* 文件上传成功函数 
		* @param res{object}   返回信息
		* @param file{object}  文件信息
		*/
		checkFile(res,file){    //发送请求，校验device文件内容 
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
				vm.showImportCard = false;
				//vm.$refs.ctableDevice.refresh(); 
				$('#tableHomeCpeList').datagrid('reload');//刷新列表
				vm.closeFileSelect();
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
			vm.selectFlag = false;
			const typeFlag = file.name.substr(file.name.lastIndexOf("."))  === '.xlsx' || file.name.substr(file.name.lastIndexOf("."))  === '.csv'
			vm.typeFlag = typeFlag;
			var groudId = '';
			if(typeFlag){
				vm.fileName = file.name;					
			}else {
				vm.fileName = '';
			}
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
			vm.showImportBox = false; //?? 无用
			vm.fileName = '';
			vm.typeFlag = true;
			vm.selectFlag = false;
			
			vm.$refs.upload.clearFiles();
		},
		// 导出设备模板
	    exportTemplate(){
	    	
	    	//cpe的设备模板导出
	    	var vm = this;
    	    var params = {};
    	    var url = "${ctx}/cell/CPE/downloadImportCpeTemplate.action";

		    params.group_id = vm.defaultGroupId;//应按照默认设备组id 还是当前选中的设备组id
	    	params.search_text = ''; //此参数是否有效
    	    params.like_fields = "serial_number";
        	var bool = checkParams(params)
			if(!bool) return false;
        	exportByForm(url,params)
	    },
		// 导入设备文件确定
		uploadDevice(){
			var vm = this;
			vm.fileParams.FileName = vm.fileName;
			vm.fileParams.group_id = vm.importGroupId;
			if(vm.fileParams.FileName){
				vm.$refs.upload.submit();
			}else{
				vm.typeFlag = true;
				vm.selectFlag = true;
			}
			
		},
	},
	mounted() {
		this.addDeviceInit();
	}
});

</script>

<script>
	var cpe_column = [];
	if(!isLWAEnable) {
		$('#lte_turbo_enable, #lte_turbo_disable').hide();
	}
	var exVm = Vue.getInstance('#cpe_export_op');
	exVm && exVm.remove();
	new Vue({
		el: '#cpe_export_op',
		methods: {
			showExport() {
				addOrImport.showAddDeviceCard = false;
				addOrImport.showImportCard = false;				
				$('.export-content').html('');
				$('.export-content').each(function(idx,item){
					if($(item).is(':visible')) {
						$(item).load('/cell/CPE/toCpeExportConfig.action',function(html){
						})
					}
				})
			}
		},
		mounted() {
			$('#enb_export_form').remove();
		}
	});
	
	var queryCpeVue = new Vue({
		el: '#toolbar_tableHomeCpeList',
		data() {

			return {
				deviceCol: [
					{code: 'SERIAL_NUMBER', label: '<%=rb.getString("CPEBianMa")%>',disabled: true},
					{code: 'CPE_NAME', label: '<%=rb.getString("CPEName")%>',disabled: true},
					{code: 'MODEL_NAME', label: '<%=rb.getString("ChanPinXingHao")%>',disabled: true},
					{code: 'PRODUCT', label: 'Module'},
					{code: 'SOFTWARE_VERSION', label: '<%=rb.getString("CPEVersion")%>',disabled: true},
					{code: 'UPTIME', label: '<%=rb.getString("YunXingShiJian")%>'},
					{code: 'first_online_time', label: '<%=rb.getString("DiYiCiLianJieShiJian")%>'},
					{code: 'LASTINFORMTIME', label: '<%=rb.getString("ShangCiLianJieShiJian")%>'},
					{code: 'MCC', label: 'MCC'},
					{code: 'MNC', label: 'MNC'},
					{code: 'group_name', label: '<%=rb.getString("SheBeiZu")%>',disabled: true},
					{code: 'module_name', label: '<%=rb.getString("MoKuaiMingCheng")%>'},
					{code: 'module_version', label: '<%=rb.getString("MoKuaiBanNen")%>'},
					{code: 'LGW_IP', label: '<%=rb.getString("LgwIPAddress")%>'}
				],
				lteCol: [
					{code: 'IMSI', label: 'IMSI',disabled: true},
					{code:'SCANMODE',label:'<%=rb.getString("SaoMiaoFangShi")%>'},
					{code: 'PCI', label: 'PCI',disabled: true},
					{code: 'HOST_NAME', label: '<%=rb.getString("HostName")%>',disabled: true},
					{code: 'CELL_IDENTITY', label: 'ECI',disabled: true},
					{code: 'DL_EARFCN', label: '<%=rb.getString("PinDian")%>'},
					{code: 'BANDWIDTH', label: '<%=rb.getString("DaiKuan")%>(MHz)'},
					{code: 'CINR0', label: 'CINR1'},
					{code: 'CINR1', label: 'CINR2'},
					{code: 'CPE_SINR', label: 'SINR'},
					{code: 'DL_CURRENT_DATARATE', label: '<%=rb.getString("CPEXiaXingTunTuLiang")%>'},
					{code: 'UL_CURRENT_DATARATE', label: '<%=rb.getString("CPEShangXingTunTuLiang")%>'},
					{code: 'TX_POWER', label: '<%=rb.getString("CPETxPower")%>'},
					{code: 'RSRP0', label: 'RSRP1'},
					{code: 'RSRP1', label: 'RSRP2'},
					{code: 'UL_MCS', label: 'UL_MCS'},
					{code: 'DL_MCS', label: 'DL_MCS'}
				],
				lanCol: [
					{code: 'MACADDRESS', label: '<%=rb.getString("CPEMacAddress")%>',disabled: true},
					{code: 'IPADDRESS', label: '<%=rb.getString("CPEIPAddress")%>',disabled: true},
					{code: 'LGW_MAC', label: '<%=rb.getString("LgwMacAddress")%>'}
				],
				locationCol: [
					{code: 'longitude', label: '<%=rb.getString("JingDu")%>'},
					{code: 'latitude', label: '<%=rb.getString("WeiDu")%>'},
					{code: 'height', label: '<%=rb.getString("GaoDu")%>'},
					{code: 'distance', label: '<%=rb.getString("JuLi")%>'},
					{code: 'link_condition', label: '<%=rb.getString("LianJieTiaoJian")%>'}
				],
				form: {
					device: ['SERIAL_NUMBER','CPE_NAME','MODEL_NAME','SOFTWARE_VERSION','group_name'],
					lte: ['IMSI','PCI','HOST_NAME','CELL_IDENTITY'],
					lan: ['MACADDRESS','IPADDRESS'],
					location: []
				},
				expanded: {
					device: true,
					lte: true,
					lan: true,
					location: true
				},
				cpeMonitorForm:{
					connection_status:[],
					cpeMonitorModule:'',
					product_model:[],
					software_version:[],
					group_id:[]
				},
				queryOptions:[
					{label:'<%=rb.getString("QuanBu")%>',value:""},
					{label:'<%=rb.getString("CPEBianMa")%>',value:"serial_number"},
					{label:'<%=rb.getString("CPEName")%>',value:"CPE_NAME"},
					{label:'<%=rb.getString("HostName")%>',value:"HOST_NAME"},
					{label:'IMSI',value:"IMSI"},
					{label:'<%=rb.getString("IPDiZhi")%>',value:"ipaddress"},
					{label:'<%=rb.getString("MACDiZhi")%>',value:"macaddress"},
					{label:'ECI',value:"CELL_IDENTITY"},
					{label:'PCI',value:"pci"},
					{label:'LGW IP',value:"lgw_ip"},
					{label:'LGW MAC',value:"lgw_mac"}
					],
				queryValue:"",
				divHeight:'30px',
				onlineOptions:[
					{label:"<%=rb.getString("QuanBu")%>",value:""},
					{label:"<%=rb.getString("LianJieZhengChang")%>",value:"1"},
					{label:"<%=rb.getString("LianJieDuanKai")%>",value:"0"},
					{label:"<%=rb.getString("TongBuZhong")%>",value:"3"},
					{label:"<%=rb.getString("TongBuShiBai")%>",value:"2"}
				],
				softwareOptions:[],
				productModelOptions:[],
				moduleOptions:[],
				deviceGroupOptions:[],
				setTop:'105px',
				search_text: '',
				showSelectButton:false,
				operator_code:operator_code,
				slideIcon:'el-icon-circle-down',
				queryParams:{
					TimeZone : timeZone
				},
				queryTip:'<%=rb.getString("CPEBianMa")%>/ <%=rb.getString("CPEName")%>/ <%=rb.getString("HostName")%>/ IMSI/ IP/ MAC/ ECI/ PCI/ LGW IP/ LGW MAC'
			};
		},
		computed: {
			colAll() {
				var vm = this;
				return vm.deviceAll && vm.lteCol && vm.lanCol && vm.locationAll;
			},
			deviceAll() {
				var vm = this;
				return vm.deviceCol.length == vm.form.device.length;
			},
			lteAll() {
				var vm = this;
				return vm.lteCol.length == vm.form.lte.length;
			},
			lanAll() {
				var vm = this;
				return vm.lanCol.length == vm.form.lan.length;
			},
			locationAll() {
				var vm = this;
				return vm.locationCol.length == vm.form.location.length;
			},
			showCols() {
				var vm = this;
				return vm.form.device.concat(vm.form.lte).concat(vm.form.lan).concat(vm.form.location);
			},
			clientWidth(){
				var val = document.body.clientWidth;
				if(val < 1440){
					return true;
				}else{
					return false;
				}
			}
		},
		methods: {
			colAllChange(val) {
				var vm = this;

				vm.deviceAllChange(val);
				vm.lteAllChange(val);
				vm.lanAllChange(val);
				vm.locationAllChange(val);
			},
			deviceAllChange(val) {
				var vm = this,
					fields = vm.deviceCol.map(function(item){
						return item.code;
					}),
					filters = vm.deviceCol.filter(function(item){
						return item.disabled;
					}).map(function(item){
						return item.code;
					});
				
				vm.form.device = val?fields:filters;
			},
			lteAllChange(val) {
				var vm = this,
					fields = vm.lteCol.map(function(item){
						return item.code;
					}),
					filters = vm.lteCol.filter(function(item){
						return item.disabled;
					}).map(function(item){
						return item.code;
					});
				
				vm.form.lte = val?fields:filters;
			},
			lanAllChange(val) {
				var vm = this,
					fields = vm.lanCol.map(function(item){
						return item.code;
					}),
					filters = vm.lanCol.filter(function(item){
						return item.disabled;
					}).map(function(item){
						return item.code;
					});
				
				vm.form.lan = val?fields:filters;
			},
			locationAllChange(val) {
				var vm = this,
					fields = vm.locationCol.map(function(item){
						return item.code;
					}),
					filters = vm.locationCol.filter(function(item){
						return item.disabled;
					}).map(function(item){
						return item.code;
					});
				
				vm.form.location = val?fields:filters;
			},
			closeConfig() {
				$(".showHideItem").slideUp(500);
			},
			columnConfig() {
				var vm = this;

				vm.configTable();
				initCPEFilterStatus();

				// 保存显示列
				var params = {
					showCol: vm.showCols.join(',')
				};

				axios.post('${ctx}/cell/CPE/cpeColumnConfig.action',stringify(params)).then(function(res){

				}).catch(function(){});

				vm.closeConfig();
			},
			configTable() {
				var vm = this;

				var columns = cpe_column.filter(function(item){
					return vm.showCols.includes(item.field) || ['cpe_operation','CONNECTION_STATUS'].includes(item.field);
				});

				if(isBatchable()) {
					columns.unshift({field:'ck',checkbox: true});
				}

				var fitColumn = false,
					main_width = $("#mainpage").width();

				if((main_width<1280 && columns.length<=5 )|| (main_width>=1280 && columns.length<=16)){
					fitColumn = true;
				}
				
				$('#tableHomeCpeList').datagrid({
					url: '${ctx}/cell/CPE/queryCpeInfosList.action?type=0',
					fitColumns: fitColumn,
					columns: [columns]
				});
			},
			init(){
				var vm = this;
				axios.post("${ctx}/cell/CPE/getCpeSelectFilter.action",stringify({
					"operator_codes":vm.operator_code
				})).then(function(response){
					var data = response.data;
					if (data) {
						vm.softwareOptions = data;
					}
					
				})
				axios.post("${ctx}/cell/CPE/getCpeSelectFilter.action",stringify({
					"selectType":"product_model",
					"operator_codes":vm.operator_code
				})).then(function(response){
					var data = response.data;
					if ( data ){
						vm.productModelOptions = data;
					}
					
				})
				axios.post("${ctx}/cell/CPE/getCpeSelectFilter.action",stringify({
					"selectType":"cpeMonitorModule",
					"operator_codes":vm.operator_code
				})).then(function(response){
					var data = response.data;
					if ( data ){
						vm.moduleOptions = data;
					}
				})
				axios.post("${ctx}/cell/CPE/getDeviceGroup4Combobox.action",stringify({
					"operator_code":vm.operator_code
				})).then(function(response){
					var data = response.data;
					if ( data ){
						vm.deviceGroupOptions = data;
					}
				})
				
				
			},
			slidedown(){
				this.divHeight = this.divHeight =='30px'?'auto':'30px';
				var height = $(".selectContent").height();
				if(height == 30){
					this.setTop = 105 + height + 'px';
					this.slideIcon = 'el-icon-circle-down'
				}else{
					this.setTop = '105px';
					this.slideIcon = 'el-icon-circle-up'
				}
				$(window).resize();
			},
			query(){
				var vm = this;
				vm.queryParams.search_text = this.search_text;
				if(vm.queryValue == ''){
					vm.queryParams['like_fields'] = 'HOST_NAME,CPE_NAME,macaddress,IMSI,serial_number,ipaddress,CELL_IDENTITY,pci,lgw_ip,lgw_mac';
				}else{
					vm.queryParams['like_fields'] = vm.queryValue;
				}
				doSearchUrl('tableHomeCpeList', vm.queryParams, '${ctx}/cell/CPE/queryCpeInfosList.action?type=0');
			},
			resetCPEQuery(){
				var vm = this;
				vm.$refs.cpeMonitorForm.resetFields();
				Object.assign(vm.queryParams,vm.cpeMonitorForm);
				doSearchUrl('tableHomeCpeList', vm.queryParams, '${ctx}/cell/CPE/queryCpeInfosList.action?type=0');
			},
			changeQuery(val){
				var tipObj = {
						'' : '<%=rb.getString("CPEBianMa")%>/ <%=rb.getString("CPEName")%>/ <%=rb.getString("HostName")%>/ IMSI/ IP/ MAC/ ECI/ PCI/ LGW IP/ LGW MAC',
						'serial_number' : '<%=rb.getString("CPEBianMa")%>',
						'CPE_NAME' : '<%=rb.getString("CPEName")%>',
						'HOST_NAME' : '<%=rb.getString("HostName")%>',
						'IMSI' : 'IMSI',
						'ipaddress' : 'IP',
						'macaddress' : 'MAC',
						'CELL_IDENTITY' : 'ECI',
						'pci' : 'PCI',
						'lgw_ip' : 'LGW IP',
						'lgw_mac' : 'LGW MAC'
				}
				this.queryTip = tipObj[val]
			},
			changeStatus(type){
				var vm = this;
				var data = vm.cpeMonitorForm[type];
				if(typeof data == 'string'){
					vm.queryParams[type] = data;
				}else{
					if(data.includes('')){
						vm.queryParams[type] = '';
					}else{
						vm.queryParams[type] = data.toString();
					}
				}
				globalQueryParams = $.extend({},vm.queryParams);
				doSearchUrl('tableHomeCpeList', vm.queryParams, '${ctx}/cell/CPE/queryCpeInfosList.action?type=0');
			}
		},
		created() {
			var vm = this,
				map = {
					device: vm.deviceCol.map(function(item){ return item.code;}),
					lte: vm.lteCol.map(function(item){ return item.code;}),
					lan: vm.lanCol.map(function(item){ return item.code;}),
					location: vm.locationCol.map(function(item){ return item.code;})
				},
				showCols = '${showCol}'.split(',');
			
			showCols.map(function(col){
				['device','lte','lan','location'].map(function(code){
					if(map[code].includes(col) && !vm.form[code].includes(col)) vm.form[code].push(col);
				});
			});

			vm.$nextTick(function(){
				vm.configTable();
			});
		},
		mounted() {
			this.init();
			closeLoading();
		}
	});

	var oschart = echarts.init(document.querySelector('#online_status_chart')),
		modulechart = echarts.init(document.querySelector('#module_name_chart')),
		devicechart = echarts.init(document.querySelector('#device_time_chart'));

	// 初始化时间轴切换事件
	var cpeChartCurrentIndex = 6;
	oschart.on('timelinechanged',function(p){
		cpeChartCurrentIndex = p.currentIndex;
		proccessCPE('online_status_chart',p.currentIndex,true);
	});

	proccessCPE('online_status_chart',cpeChartCurrentIndex,false);
	var cpeLayerTimer = setInterval(function(){
		var ctn = document.querySelector('#cpeInfo');
		
		if(!ctn) {
			clearInterval(cpeLayerTimer);
			return;
		}

		var cls = ctn.classList;

		if(cls.contains('expanded')) {
			proccessCPE('online_status_chart',cpeChartCurrentIndex,true);
		}else {
			proccessCPE('online_status_chart',6,false);
		}
		resieChart();
	},1000*60*10);
	
	initCpeCharts(false);

	window.addEventListener('resize',debounce(resieChart));

	function initCpeCharts(expanded) {
		initCpeModuleName(expanded);
		initCpeDeviceRunTime(expanded);
	}

	function initCpeModuleName(expanded) {
		var modulechart = echarts.init(document.querySelector('#module_name_chart')),
			legend = [],
			splitNum = 10,
			colorList = ['#69b6fc','#90ec97','#e9a4a4','#f3cc90','#d7a3ef','#ada3ef','#4bdedb','#4bdea3','#e6a46b','#efa3d3','#f5f5f5'],
			sdata = [];

		$.post('${ctx}/system/device/getCpeModuleStatisticsData.action',function(data){
			if(data) {
				var total = 0,
					others = [];
				data.map(function(item){
					total += item.device_count*1;
				});
				data.map(function(item,index) {
					var name = item.module_name + ': '+item.device_count + ' (' + (item.device_count*100/total).toFixed(2) + '%)';
					if(index<splitNum) {
						legend.push(name);
						sdata.push({name: name, value: item.device_count*1});
					}else {
						others.push({name: name, value: item.device_count*1});
					}
				})

				if(others.length) {
					var otherCount = 0;
					others.map(function(item){
						otherCount += item.value;
					});
					var otherName = '<%=rb.getString("QiTa")%>: '+otherCount+ ' (' + (otherCount*100/total).toFixed(2) + '%)';
					legend.push(otherName);
					sdata.push({name: otherName, value: otherCount});
				}
				
				colorList = colorList.slice(0,splitNum);
				colorList.push('#f5f5f5');
			}
			
			modulechart.setOption({
				title: {
					top: 10,
					left: 10,
					text: '<%=rb.getString("ChanPinXingHao")%>',
					textStyle: {
						fontSize: 12
					}
				},
				color: colorList,
				grid: {
					top: 60,
					left: 35,
					right: 15,
					bottom: 90
				},
				tooltip: {
					trigger: 'item',
					formatter: '{b}'
				},
				legend: {
					itemWidth: expanded?15:10,
					orient: expanded?'vertical':'horizontal',
					top: expanded?60:null,
					right: expanded?5:null,
					bottom: expanded?null:5,
					itemHeight: expanded?10:5,
					textStyle: {
						fontSize: expanded?10:6
					},
					data: legend
				},
				series: [
					{
						name: '',
						type: 'pie',
						radius: '50%',
						center: expanded?['35%','50%']:['45%','40%'],
						data: sdata,
						emphasis: {
							itemStyle: {
								shadowBlur: 10,
								shadowOffsetX: 0,
								shadowColor: 'rgba(0,0,0,0.5)'
							}
						},
						label: {
							normal: {
								show: expanded,
								color: '#333',
								fontSize: 10
							}
						},
						labelLine: {
							normal: {
								show: expanded
							}
						}
					}
				]
			});
		},'json');
	}

	function initCpeDeviceRunTime(expanded) {
		var devicechart = echarts.init(document.querySelector('#device_time_chart')),
			legend = ['<10days','10-30days','30-90days','>90days'],
			data = [];

		$.post("${ctx}/system/device/getCpeUpTimeRealData.action", function(data){
			if(data) {
				data = [
					data.tenDayDeviceCount,
					data.thirtyDayDeviceCount,
					data.ninetyDayDeviceCount,
					data.bigNinetyDayDeviceCount
				];
				var total = (data[0]+data[1]+data[2]+data[3])||1,
					lgd1 = '<10days: ' + data[0] + ' (' + (data[0]*100/total).toFixed(2)+'%)',
					lgd2 = '10-30days: ' + data[1] + ' (' + (data[1]*100/total).toFixed(2)+'%)',
					lgd3 = '30-90days: ' + data[2] + ' (' + (data[2]*100/total).toFixed(2)+'%)',
					lgd4 = '>90days: ' + data[3] + ' (' + (data[3]*100/total).toFixed(2)+'%)';

				legend = [
					lgd1,
					lgd2,
					lgd3,
					lgd4
				];
				series = [
					{
						name: lgd1,
						type: 'bar',
						data: []
					},
					{
						name: lgd2,
						type: 'bar',
						data: []
					},
					{
						name: '<%=rb.getString("ChiXuShiChang")%>',
						type: 'bar',
						data: data,
						barWidth: expanded?40:15,
						itemStyle: {
							normal: {
								color: function(param){
									return ['#69b6fc','#90ec97','#e9a4a4','#f3cc90'][param.dataIndex];
								},
								label: {
									show: true,
									position: 'top',
									formatter: '{c}'
								}
							}
						}
					},
					{
						name: lgd3,
						type: 'bar',
						data: []
					},
					{
						name: lgd4,
						type: 'bar',
						data: []
					}
				];
			}
			
			devicechart.setOption({
				title: {
					top: 10,
					left: 10,
					text: '<%=rb.getString("SheBeiYunXingShiChang")%>',
					textStyle: {
						fontSize: 12
					}
				},
				color: ['#69b6fc','#90ec97','#e9a4a4','#f3cc90'],
				grid: {
					top: 50,
					left: 35,
					right: 45,
					bottom: expanded?70:90
				},
				tooltip: {
					trigger: 'axis',
					formatter: function(p) {
						var ops = p[0];
						
						return legend[ops.dataIndex];
					}
				},
				legend: {
					orient: (expanded?'horizontal':'vertical'),
					itemHeight: expanded?12:8,
					left: 30,
					bottom: 5,
					data: legend,
					textStyle: {
						fontSize: expanded?12:8
					},
					icon: 'circle',
					selectedMode: false
				},
				xAxis: {
					name: '<%=rb.getString("Tian")%>',
					type: 'category',
					data: ['<10','10-30','30-90','>90'],
					axisLine: {
						show : true,
						lineStyle:{ color:"#707070" }
					},
					axisLabel: {
						show: expanded
					},
					axisTick: {
						show: expanded
					}
				},
				yAxis: {
					type: 'value',
					axisLabel : {
						show: true,
						textStyle:{ color:"#707070" },
						lineStyle:{ color:"#707070" },
						fontSize: expanded?10:8
					},
					axisLine:{
						lineStyle:{ color:"#707070" }
					},
					splitLine : {
						lineStyle:{ color:"#f1f1f4" }
					}
				},
				series: series
			});
		});
	}

	function proccessCPE(code,index,bool) {
		var vm = this,
			s_time = getYesterDay(6-index).substring(0,10)+' 00:00:00',
			e_time = getYesterDay(6-index).substring(0,10)+' 23:59:59',
			unitName='<%=rb.getString("GeShu")%>';
			
		end_time = getNowTimeToZoneTimeRange(timeZone, 144).end_time;
		
		if(index==6) {
			e_time = end_time;
		}else{
			e_time = getYesterDay(6-index-1).substring(0,10)+' 00:00:00';
		}
		var paramsCPE = {
				device_type: "cpe",
				time_level: "min",
				timeZone: timeZone,
				start_time: s_time,
				end_time: e_time
			};
		
		$.post("${ctx}/system/device/getDeviceStatisticsDataList.action", paramsCPE, function(data){
			var params = {
					legend: ['Connected','Disconnected'],
					legendNames: ['<%=rb.getString("ShouYe_LianJie")%>','<%=rb.getString("ShouYe_FeiLianJie")%>'],
					query: paramsCPE,
					data: data,
					unit: unitName,
					pointerCount: 6,
					index: index,
					code: code,
					title: 'CPE <%=rb.getString("ShouYe_LianJieZhuangTai")%>'
				};

			var dom = document.querySelector('#'+code);
			if(dom){
				var chart = echarts.getInstanceByDom(dom);
				chart.setOption(createCpeOption(params, bool)); // 根据请求的数据重新渲染图表
			}
		}, "json");
	}

	function createCpeOption(params,expanded) {
		var pointerCount = params.pointerCount, title = params.title,
			finalArr = proccessData(params), startMinDashboard = ' 00:00:00',
			index = params.index, code = params.code, unit =  params.unit,
			legend = params.legend || [], legendNames = params.legendNames,
			type1 = legend[0], type2 = legend[1],
			dates = getDays(),
			// totalMap = {
			// 	'<%=rb.getString("ShouYe_LianJie")%>': connectionStatus,
			// 	'<%=rb.getString("ShouYe_FeiLianJie")%>': connectionStatusRef
			// },
			option = {
				baseOption: {
					title: {
						text: title,
						textStyle: {
							fontSize: 12
						},
						top: 10,
						left: expanded?20:10
					},
					timeline: {
						show: expanded,
						axisType: 'category',
						controlPosition: 'none',
						symbolSize:8,
						lineStyle : { color : '#B0AFBA',width : 1 },
						itemStyle : {
							normal : { borderColor : '#B0AFBA' },
							emphasis : {
								borderColor : '#1e90ff',
								color : '#1e90ff'
							}
						},
						data: [getYesterDay(6).substring(5).replace("-","."),
								getYesterDay(5).substring(5).replace("-","."),
								getYesterDay(4).substring(5).replace("-","."),
								getYesterDay(3).substring(5).replace("-","."),
								getYesterDay(2).substring(5).replace("-","."),
								getYesterDay(1).substring(5).replace("-","."),
								getYesterDay(0).substring(5).replace("-",".")],
						notMerge:true,
						currentIndex: index,
						checkpointStyle:{
							color:'#209FFF',
							borderColor:'none'
						}
					},
					tooltip : {
						trigger : 'axis',
						formatter:function(params){
							var timeStr = "";
							var params = JSON.parse(JSON.stringify(params));
							var str = "";
							if(code == 'ueCount'){
								if(params[0]){
									str += "<div>" + params[0].name + "</div>";
									str += "<div>" + params[0].seriesName + ": " + params[0].data + "</div>" 
								}
							}else{
								for(var i=0;i<params.length;i++){
									if (params[0].name.indexOf("/")>=0){
											timeStr = params[0].name.split("/");
											if(!str){
												str += "<div>" + timeStr[0] + " -- </div>";
												str += "<div>" + timeStr[1] + "</div>";                     
											}
										} else {
											if(str == ''){
												str += "<div>" + params[0].name + "</div>";
											}
										}
									str += "<div>" + params[i].seriesName + ": " + params[i].data + "</div>" 
								}
							}
							return str;
						}
					},
					grid:{
						bottom: expanded?70:20,
						left:  expanded?40:20
					}, 
					color: ['#90EC97','#E9A4A4'],
					legend: {
						top: expanded?10:30,
						itemHeight: expanded?12:8,
						textStyle: {
							fontSize: expanded?12:8
						},
						data: legendNames,
						// formatter: function(name) {
						// 	return name+'('+totalMap[name]+')';
						// }
					},
					xAxis : [{
						name : expanded?'<%=rb.getString("XiaoShi")%>':'',
						type : 'category',
						boundaryGap : false,
						axisLine:{
							show : true,
							lineStyle:{ color:"#707070" }
						},
						axisLabel : {
							show: expanded,
							textStyle:{ color:"#707070" },
							lineStyle:{ color:"#707070" },
							formatter : function(val) {
								var secondTime = val.split(' ')[1];
								clock = secondTime.substring(0,2);
								val = secondTime.substring(0,5);
								if(secondTime.substring(3,5)=='00') return clock;
								return val;
							},
							interval : function(index){
								if(index%pointerCount == 0 && index != pointerCount*24){
									return true;
								}
							},
							rotate : (function(){
									var degree = 0;
									if(startMinDashboard != " 00:00:00"){
										degree = 45;
									}
									return degree;
							})(),
						},
						axisTick: {
							show: expanded
						}
					}],
					yAxis : [{
						name: expanded?unit:'',
						minInterval: (code=='thoughtput' || code=='traffic' )? null:1,
						type: 'value',
						axisLabel : {
							show:true,
							textStyle:{ color:"#707070" },
							lineStyle:{ color:"#707070" }
						},
						axisLine:{
							lineStyle:{ color:"#707070" }
						},
						splitLine: {
							lineStyle:{ color:"#f1f1f4",type: 'dotted'}
						}
					}],
					series : [{  
						name: legendNames[0],
						type: 'line',
						symbolSize: 1,
						itemStyle: {
							normal: {
								areaStyle: {opacity: 0.2}
							}
						},
						areaStyle: {opacity: 0.2},
						showAllSymbol: true,
						step: false,
						connectNulls: true
					},
					{  
						name: legendNames[1],
						type: 'line',
						symbolSize: 1,
						itemStyle: {
							normal: {
								areaStyle: {opacity: 0.2}
							}
						},
						areaStyle: {opacity: 0.2},
						showAllSymbol : true,
						step: false,
						connectNulls: true
					}]
				},
				options: []
			};
		
		var subOpts = [];
		for(var idx=0; idx<7; idx++) { // 初始化空数据
			subOpts.push({  
							xAxis: [{ data: []}],
							series: [
								{data: []} , 
								{data: []}
							]  
						});
		}
		option.options = subOpts;
		// 设置当前选中的日期节点数据
		option.options[index].xAxis[0].data = finalArr[getYesterDay(6-index)]['x'];
		option.options[index].series[0].data = finalArr[getYesterDay(6-index)][type1];
		option.options[index].series[1].data = finalArr[getYesterDay(6-index)][type2];

		var sm1 = option.options[index].series[0].data.some(function(item){
				return item!='-';
			}),
			sm1 = option.options[index].series[1].data.some(function(item){
				return item!='-';
			})
		if(sm1 && sm1) {
			option.options[index].yAxis = [{min: null, max: null}];
			document.querySelector('#online_status_chart').classList.remove('no-data');
		}else {
			option.options[index].yAxis = [{min: 0, max: 3}];
			document.querySelector('#online_status_chart').classList.add('no-data');
		}
		
		return option;
	}

	function proccessData(params) {
		var startMinDashboard = ' 00:00:00',
			pointerCount = params.pointerCount,
			chart_data = params.data, chart_id = params.code,
			deviceCount = "", pointerNum = pointerCount*24+1,
			legend_code = params.legend, legend_name = legend_code,
			legend_code_line1 = legend_code[0], legend_code_line2 = legend_code[1],
			type1 = legend_code[0], type2 = legend_code[1];

		dataArr=[];
		$.each(chart_data,function(index,item){
			var xDate = item.statistics_time.split(' ');
			deviceCount = item.device_count;
			var onlineOrActiveOrConnectCount = "";    
			var offOnlineOrInactiveOrDisconnectCount = "";
			if(chart_id == 'online_status_chart'){
				onlineOrActiveOrConnectCount = item.online_count;
				offOnlineOrInactiveOrDisconnectCount = deviceCount - onlineOrActiveOrConnectCount;
				dataArr[index]=[xDate[0],xDate[1],onlineOrActiveOrConnectCount,offOnlineOrInactiveOrDisconnectCount];
			}
		});
		finalArr = [];//[2017-01-01,00:00:00,0,1,'25' ]
		for(var initIndex=0;initIndex<7;initIndex++ ){
			var arrIndex = getYesterDay(initIndex);
			if(!finalArr[arrIndex]){
				var activeCountArr = [], yaxisArr = [],onlineOrActiveOrConnectCountArr=[],offOnlineOrInactiveOrDisconnectCountArr=[];
				for(var axisIndex=0; axisIndex<pointerNum; axisIndex++){
					var yaxisStartTime = getYesterDay(initIndex).substring(0,10)+ startMinDashboard,
						offsetTime = addTimes(new Date(yaxisStartTime),axisIndex*(60/pointerCount));
					yaxisArr.push(formatDate(offsetTime));
					onlineOrActiveOrConnectCountArr.push('-');
					offOnlineOrInactiveOrDisconnectCountArr.push('-');
				}
				finalArr[arrIndex] = [];
				finalArr[arrIndex]['x'] = yaxisArr; //x轴数据
				finalArr[arrIndex][type1] = onlineOrActiveOrConnectCountArr;
				finalArr[arrIndex][type2] = offOnlineOrInactiveOrDisconnectCountArr;
			}
		}
		$.each(dataArr,function(n,m){
			var dateIndex = m[0],yValue=m[1].substring(0,5),onlineOrActiveOrConnectArrValue=m[2],offOnlineOrInactiveOrDisconnectValue=m[3];
			if(finalArr[dateIndex]['x'].includes(dateIndex+' '+m[1])){
				var differTimes = new Date(dateIndex+' '+m[1]).getTime() - new Date(dateIndex+' 00:00:00').getTime();
				var axisDateIndex = Math.round(differTimes/(1000*60*(60/pointerCount)));
				finalArr[dateIndex][type1].splice(axisDateIndex,1,onlineOrActiveOrConnectArrValue); 
				finalArr[dateIndex][type2].splice(axisDateIndex,1,offOnlineOrInactiveOrDisconnectValue); 
			}
		});
		var prevDayData = "";
		for(var key in finalArr ){
			if(prevDayData){
				['Connected','Disconnected','Active','Inactive','Online','Offline'].map(function(itemCode){
					if(finalArr[key][itemCode]){
						finalArr[key][itemCode][finalArr[key][itemCode].length-1] = prevDayData[itemCode][0];
					}
				});
				prevDayData = finalArr[key]
			}else prevDayData = finalArr[key]
		}
		
		return finalArr;
	}

	function getDays() {
		var lineDays = [],
			start = new Date();
		
		for(var i = 6; i>=0; i--) {
			var dateStr = dateformatter(addDate(start,-i));
			lineDays.push(dateStr.substr(0,10));
		}
		
		return lineDays;
	}

	function resieChart() {
		[oschart,modulechart,devicechart].map(function(chart){
			chart && chart.resize();
		})
	}

	function toggleExpand() {
		var ctn = document.querySelector('#cpeInfo'),
			cls = ctn.classList;
		
		if(cls.contains('expanded')) {
			cls.remove('expanded');
			proccessCPE('online_status_chart',6,false);
			initCpeCharts(false);
		}else {
			cls.add('expanded');
			proccessCPE('online_status_chart',6,true);
			initCpeCharts(true);
		}
		resieChart();
		addOrImport.showAddDeviceCard = false;
		addOrImport.showImportCard = false;
	}
	
	function minisize() {
		
		var ctn = document.querySelector('#cpeInfo'),
			cls = ctn.classList;
		
		if(cls.contains('minisize')) {
			cls.remove('minisize');
		}else {
			cls.add('minisize');
		}
		
		resieChart();
		$('#tableHomeCpeList').datagrid('resize');
		addOrImport.showAddDeviceCard = false;
		addOrImport.showImportCard = false;
	}

var enableModifyPwd = writableMap['CODE_CPE_CHANGE_PASSWORD'],
	enableReboot = writableMap['CODE_CPE_REBOOT'],
	enableSync = writableMap['CODE_CPE_SYNCHRONIZE'],
	curCPECode = ''; // 用户兼容单选模式的行选择记录
var deviceFlag = false;
	
function isBatchable(){
	return enableModifyPwd == true || enableReboot == true || enableSync == true;
}
var columncpe = "${cpeColumn}";     //未选中标识 
var allColumn = "${allColumn}"      //上次保存的所有参数顺序 
var versionAttr = "${isQb}";
var selContent = "";
var sortcolumn = "";
var apnFalg = false;
var itemList;
var showProcessSetFlag = false;

//RSRP
var lowVal = localStorage.getItem("rsrp1");
var highVal = localStorage.getItem("rsrp2");

var no_rebootCell = "${no_rebootCell}";
/* 筛选展示逻辑处理 */
var filterUrl = '${ctx}/cell/CPE/queryFilterCpeInfosList.action';
var sortCPE = '' , orderCPE = '' ;
var globalQueryParams = {
		TimeZone : timeZone,
        // 拼装查询框模糊匹配的字段（注：要和数据库表字段一致）
        like_fields: 'HOST_NAME,cpe_name,macaddress,imsi' 
    };
var connectionStatus='';
var connectionStautsRef='';
var filterTop = 80;
$('#mainpage').mousedown(function(){/* 菜单隐藏处理 */
	try{
	    var target = event.target, list = Array.from(target.classList),
	        plist = Array.from(target.parentNode.classList);
	    if(!(list.includes('filter-menu') || list.includes('filter-item') || plist.includes('filter-item'))){
	      $('.filter-menu').hide();
	    }
	}catch(e){}
});

//$(function(){
	//closeLoading();
	$("#CPESoftVersion").combobox({
    	url: '${ctx}/cell/CPE/getCpeSelectFilter.action?operator_codes=' + operator_code,
    	panelHeight:50,
        valueField: 'value',
        textField: 'text'
    });
	$("#CPESoftVersion").combobox('setValue','');
	$("#CPEDeviceGroup").combobox({
    	url: '${ctx}/cell/CPE/getDeviceGroup4Combobox.action?operator_code=' + operator_code,
        valueField: 'value',
        textField: 'text'
    });
	$("#CPEDeviceGroup").combobox('setValue','');
	
	$(document).click(function(e){
        var e = e || window.event;
        var elem = e.target || e.srcElement;
		
		$('.version-details').each(function(idx, item){
			if(elem.contains(item)) $(item).fadeOut();
		});

        while(elem){
            if($(elem).hasClass('el-icon-operation-more') || elem.className == 'showCPEOp' || elem.className == 'slideDiv'){
                return
            }
            elem = elem.parentNode;
        }
        if($(e.target).closest(".window-mask").length==0
      			 &&$(e.target).closest(".messager-window").length==0
      			 &&$(e.target).closest("#cpeMoreQueryImg").length==0&&$(e.target).closest("#cpeHiddenSpan").length==0
      			 &&$(e.target).closest(".searchResultImgChangeStyle").length==0
      			 &&$(e.target).closest("#cpeQueryDiv").length==0&&$(e.target).closest("#cpesummaryQueryDiv").length==0){
           	
           	$("#cpeMoreQueryImg").removeClass('expanded');
           	$("#cpeMoreQueryImg").attr("flag","1"); 
           	$("#cpeHiddenSpan").hide();
      	}
        $(".showCPEOp").hide();
        $("#cpemn").hide();
    })
	
	$(".cpe1ExportConfigItem").mousedown(function(event){
		if(event.target.tagName == "INPUT" || event.target.tagName == "LABEL" || (event.target.tagName == "SPAN" && $(event.target).hasClass('el-icon-moveTop'))){
			
		}else{
			$(this).addClass("handleMouseDown");
		}
		
	})
	$(".cpe1ExportConfigItem").mouseup(function(){
		$(this).removeClass("handleMouseDown");
	})
	
	<%-- 首页-CPE信息列表-搜索框回车事件 --%>
	$("#cpeSearchText").bind("keyup", function(e){
		if (e.keyCode == 13){
			queryCPETable('e');
		}
	});
	
	cpe_column = [
		{field:'CPE_CODE',hidden:true,title:'<%=rb.getString("XiaoZhanBianMa")%>'},
		{field:'lastsyntime',hidden:true},
		{field:'cpe_operation',fixed: true,width: 30,sortable:false,formatter:cpeTaskFormatter,styler:setStyle,title:''},
		{field:'CONNECTION_STATUS',title:'',sortable:true,width:45,formatter:connStatusFormatterSyn},
		{field:'SERIAL_NUMBER',sortable:true,width:180,title:'<%=rb.getString("CPEXuLieHao")%>'},
		{field:'CPE_CODE',hidden:true,title:'<%=rb.getString("XiaoZhanBianMa")%>'},
		{field:'CPE_NAME',editor:'text',sortable:true,width:120,title:'<%=rb.getString("CPEName")%>'},	                  
		{field:'IMSI',sortable:true,width:120,title:'IMSI'},
		{field:'MACADDRESS',sortable:true,width:120,title:'MAC'},
		{field:'IPADDRESS',sortable:true,width:110,formatter: ipAddrFormatter,title:'IP'},
		{field:'MODEL_NAME',sortable:true,formatter:capablityFormatter,width:120,title:'<%=rb.getString("ChanPinXingHao")%>'},
		{field:'SOFTWARE_VERSION',sortable:true,width:160,formatter: softwareVersionFmt,title:'<%=rb.getString("CPEVersion")%>'},
		{field:'group_name',sortable:true,width:130,title:'<%=rb.getString("SheBeiZu")%>'},
		{field:'HOST_NAME',sortable:true,width:90,title:'<%=rb.getString("HostName")%>'},
		{field:'CELL_IDENTITY',sortable:true,width:60,title:'ECI'},
		{field:'SCANMODE',width:150,title:'<%=rb.getString("SaoMiaoFangShi")%>',formatter:modeFmt},
		{field:'PCI',sortable:true,width:60,formatter:tablePciStatus,title:'PCI'},
		{field:'LGW_IP',sortable:true,width:130,formatter: lgwIPFormatter,title:'<%=rb.getString("LgwIPAddress")%>'}, 
		{field:'LGW_MAC',sortable:true,width:140,title:'<%=rb.getString("LgwMacAddress")%>'}, 
		{field:'HISTORY_DATA',sortable:false,width:50,formatter:historyGraph,title:'<%=rb.getString("LiShiShuJu")%>'},
		{field:'UL_MCS',sortable:true,width:75,title:'UL_MCS'},
		{field:'DL_MCS',sortable:true,width:75,title:'DL_MCS'},
		{field:'RSRP0',sortable:true,formatter:showRedAccordingRSRP,width:70,title:'RSRP1'},
		{field:'RSRP1',sortable:true,formatter:showRedAccordingRSRP,width:70,title:'RSRP2'},
		{field:'CINR0',sortable:true,width:70,title:'CINR1'},
		{field:'CINR1',sortable:true,width:70,title:'CINR2'},
		{field:'CPE_SINR',sortable:true,width:60,title:'SINR'},
		{field:'DL_CURRENT_DATARATE',sortable:true,width:150,title:'<%=rb.getString("CPEXiaXingTunTuLiang")%> (Mbps)'},
		{field:'UL_CURRENT_DATARATE',sortable:true,width:150,title:'<%=rb.getString("CPEShangXingTunTuLiang")%> (Mbps)'},
		{field:'UPTIME',sortable:true,width:120,title:'<%=rb.getString("YunXingShiJian")%>'},
		{field:'first_online_time',sortable:true,width:130,title:'<%=rb.getString("DiYiCiLianJieShiJian")%>'},
		{field:'LASTINFORMTIME',sortable:true,width:130,title:'<%=rb.getString("ShangCiLianJieShiJian")%>'},    
		{field:'PRODUCT',sortable:true,width:60,formatter:formatCPEType,title:'<%=rb.getString("CPELeiXing")%>'},
		{field:'TX_POWER',sortable:true,width:80,title:'<%=rb.getString("CPETxPower")%>'},
		{field:'DL_EARFCN',width:60,title:'<%=rb.getString("PinDian")%>'},
		{field:'BANDWIDTH',width:100,title:'<%=rb.getString("DaiKuan")%>(MHz)'},
		{field:'MCC',sortable:true,width:60,title:'MCC'},
		{field:'MNC',sortable:true,width:60,title:'MNC'},
		{field:'OLDPRODUCT',hidden:'true'},
		// {field:'LTE_STATUS',width:100,title:'LTE Status'},
		{field:'longitude',width:75,title:'<%=rb.getString("JingDu")%>'},
		{field:'latitude',width:70,title:'<%=rb.getString("WeiDu")%>'},
		{field:'height',width:75,title:'<%=rb.getString("GaoDu")%>'},
		{field:'distance',width:70,title:'<%=rb.getString("JuLi")%>'},
		{field:'link_condition',width:100,title:'<%=rb.getString("LianJieTiaoJian")%>'},
		{field:'module_name',width:150,title:'<%=rb.getString("MoKuaiMingCheng")%>'},
		{field:'module_version',width:150,title:'<%=rb.getString("MoKuaiBanNen")%>', formatter: moduleVersionFmt}
	];
	
	var show_column = [
			{field:'cpe_operation',fixed: true,width: 30,sortable:false,formatter:cpeTaskFormatter,styler:setStyle,title:''},
			{field:'CONNECTION_STATUS',title:'',sortable:true,width:45,formatter:connStatusFormatterSyn},
			{field:'SERIAL_NUMBER',sortable:true,width:180,title:'<%=rb.getString("CPEXuLieHao")%>'},
			{field:'CPE_CODE',hidden:true,title:'<%=rb.getString("XiaoZhanBianMa")%>'},
			{field:'CPE_NAME',editor:'text',sortable:true,width:120,title:'<%=rb.getString("CPEName")%>'},
			{field:'IMSI',sortable:true,width:120,title:'IMSI'},
			{field:'MACADDRESS',sortable:true,width:120,title:'MAC'},
			{field:'IPADDRESS',sortable:true,width:110,formatter: ipAddrFormatter,title:'IP'},
			{field:'MODEL_NAME',sortable:true,formatter:capablityFormatter,width:120,title:'<%=rb.getString("ChanPinXingHao")%>'},
			{field:'SOFTWARE_VERSION',sortable:true,width:160,formatter: softwareVersionFmt,title:'<%=rb.getString("CPEVersion")%>'},
			{field:'group_name',sortable:true,width:130,title:'<%=rb.getString("SheBeiZu")%>'},
			{field:'HOST_NAME',sortable:true,width:90,title:'<%=rb.getString("HostName")%>'},
			{field:'CELL_IDENTITY',sortable:true,width:60,title:'ECI'},
			{field:'PCI',sortable:true,width:60,formatter:tablePciStatus,title:'PCI'}
		];
	
	if(isBatchable()) {
		show_column.unshift({field:'ck',checkbox: true});
	}
	
	var fitColumn = true;
	
	$("#tableHomeCpeList").datagrid({
		border : false,
		fit : true,
		//url : '${ctx}/cell/CPE/queryCpeInfosList.action?type=0',
		queryParams : {
			isCloudCore: isCloudCore,
			TimeZone : timeZone,
			// 拼装查询框模糊匹配的字段（注：要和数据库表字段一致）
			like_fields : 'HOST_NAME,cpe_name,macaddress,imsi,serial_number'
		},
		toolbar : '#toolbar_tableHomeCpeList',
		singleSelect : !isBatchable(),
		rownumbers : true,
		fitColumns : fitColumn,
		pageSize : 50,
		pageList : [50,100,200],
		pagination : true,
		pagePosition : 'bottom',
		striped : true,
		idField : 'CPE_CODE',
		columns : [show_column],
		checkOnSelect: false,
		selectOnCheck: true,
		onBeforeLoad : tableHomeCpeListBeforeLoad,
		//onRowContextMenu : showRowMenutableHomeCpeList, 
		onLoadError : datagridLoadError,
		onLoadSuccess : cpeMonitorDatagridLoadSuccess,
		autoSize: false,
		onBeforeSelect:onBeforeSelect,
		onCheck: function(index,row){
			refreshSelectedEnbList();
		},
		onUncheck: function(index,row){
			refreshSelectedEnbList();
		},
		onCheckAll: function(rows){
			refreshSelectedEnbList();
		},
		onUncheckAll: function(rows){
			refreshSelectedEnbList(true);
		},
		onSortColumn : function(sort , order){
			sortCPE = sort;
			orderCPE = order;
		}
	})
	
	$("#kaiqi").bind("change",function(){
		//移除之前被选中的元素
		var isChecked = $("#kaiqi").is(":checked");
		if(true==isChecked){
			$("#kaiqi").val("1");
		}else{
			$("#kaiqi").val("0");
		}
	});
	// 初始化密码修改窗口
	$('#modify_password_dialog').dialog({
		modal: true,
		closed: true,
		width: 400,
		height: 240
	}).parents('.window').addClass('beautify');
	
	$('#pwd_type_change').off('click').on('click',function(){
		var icon = $(this);
		if(icon.hasClass('el-icon-operation-hide')) {
			$('#pwd_input').attr('type','text');
		}else {
			$('#pwd_input').attr('type','password');
		}
		icon.toggleClass('el-icon-operation-hide').toggleClass('el-icon-operation-view');
	});


	
//});
var cpetopoLoaded = false;
function loadCpeTopo() {
	if(cpetopoLoaded) return;
	cpetopoLoaded = true;
	$('#cpe_topo_ctner').load('${ctx}/cell/topo/toCPETopo.action',function(html){
		$.parser.parse(this);
	});
}

function showFixed() {
	$('#cpeInfo').removeClass('hide-fixed');
	resieChart();
}

function hideFixed() {
	$('#cpeInfo').addClass('hide-fixed');
}

function ipAddrFormatter(value,row,index) {
	/* if(value) {
		value = '<a href="' + value + '" target="_blank" style="color: #4d84ff;">'+value+'</a>';
	} */

	return value;
}
function lgwIPFormatter(value,row,index) {
	if(value) {
		value = '<a href="' + row.lgw_url + '" target="_blank" style="color: #4d84ff;">'+value+'</a>';
	}

	return value;
}
//批量LTE开关
function ltmkai(statu){
	var checkedRow = $("#tableHomeCpeList").datagrid("getChecked");
	var cpeCodes = '';
	checkedRow.map(function(item){
		cpeCodes += item.CPE_CODE + ',';
	})
	var params = {
		turboEnable : statu,
		cpeCodes : cpeCodes.substring(0,cpeCodes.length-1)
	}
	let Msg = ''
	if(statu=="0"){
		Msg = "<%=rb.getString("PiLiangGuanCapacity")%>"
	}else{
		Msg = "<%=rb.getString("PiLiangKaiCapacity")%>"
	}
	$.messager.confirm('<%=rb.getString("QueRen")%>', Msg, function (r) {
			if (r) {
				$.post("${ctx}/cell/ap/turboSetting.action",params,function(data){
					if(data["success"]){
						showMsg('success_msg','<%=rb.getString("ChengGong")%>');
						$("#tableHomeCpeList").datagrid("reload");
						delAllSelectedRecord();
					}else{
						showMsg('error_msg',data["message"]);
						//showMsg("error_msg",data["message"]);
					}
				},"json")
			}
		}).addClass('normalConfirm');
	
	
}
// 打开密码修改层
function toModifyPassword(){
	$('#modify_password_dialog').dialog('open').dialog('vcenter');
	$('#modify_password_dialog').removeClass('modified');
}
//批量同步
function cpeRefreshBatch(){
	var checkCpe = $("#tableHomeCpeList").datagrid("getChecked");
	var selCpe,
		cpeCodes = '';
	checkCpe.map(function(item){
		selCpe = item;
		cpeCodes += selCpe.CPE_CODE + ',';
		var rowIndex=$("#tableHomeCpeList").datagrid("getRowIndex",selCpe.CPE_CODE);
		if(selCpe.CONNECTION_STATUS != 'Off'){
			selCpe.CONNECTION_STATUS = 'updating';
			$("#tableHomeCpeList").datagrid("updateRow",{
				index:rowIndex,
				row:selCpe
			});
		}
	})
	var params = {
		cpe_type : 0,
		cpeCode : cpeCodes.substring(0,cpeCodes.length-1)
	}
	$.post("${ctx}/cell/CPE/refreshCpeInfo.action",params,function(data){
		if(!data["success"]){
			showMsg("error_msg",data["message"]);
		}else{
			delAllSelectedRecord();
		}
	},"json")
}
//批量重启
function cpeRebootBatch(){
	var checkedRow = $("#tableHomeCpeList").datagrid("getChecked");
	var cpeCodes = '';
	checkedRow.map(function(item){
		cpeCodes += item.CPE_CODE + ',';
	})
	var params = {
		cpeCodes : cpeCodes.substring(0,cpeCodes.length-1)
	}
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingChongQiSheBei")%>", function (r) {
		if (r) {
			var param = {
					cpeCodes: cpeCodes
			};
			$.post("${ctx}/cell/CPE/rebootCpe.action", param, function (data) {
				if (data["success"]) {
					delAllSelectedRecord();
				}else{
					showMsg('error_msg',data["message"]);
				}
			}, "json");
		}
	}).addClass("seriousConfirm");
}
// 校验密码
function validatePasswordFun() {
	var value = $('#pwd_input').val(),
		msg = '',
		errorTip = $('#pwd_error'),
		valid = true;
	if(value) {
		if(value.length<5 || value.length>15) { // 长度不符合 5 - 15位
			msg = '<%=rb.getString("ZiFuChang")%><%=rb.getString("MaoHao")%> 5-15 <%=rb.getString("ZiFuFuShu")%>' ;
		}else {
			var reg = /[\u4E00-\u9FA5\uF900-\uFA2D]/; // 中文校验
			if(reg.test(value)) {
				msg = '<%=rb.getString("FeiZhongWenZiFu")%>';
			}
		}
	}else {
		msg = '<%=rb.getString("QingShuRuMiMa")%>';
	}
	if(msg) {
		valid = false;
	}
	errorTip.text(msg);
	
	return valid;
}
// 确认修改密码
function changepwdFun() {
	if(!validatePasswordFun()) return;
	
	$('#modify_password_dialog').addClass('modified');
	
	var tb = $('#tableHomeCpeList'),
		rows = tb.datagrid('getChecked'),
		devices = '',
		newPwd = $("#pwd_input").val();
	if(rows) {
		devices = rows.map(function(row){
			return row.CPE_CODE
		}).join(',')
	}
	var params = {
			timeZone: timeZone,
			password: newPwd,
			executeType: 'active',
			devices: devices,
			fromCpeMonitor: 'yes'
		};
	$.post("${ctx}/task/cpe/changepwd/addTask.action", params, function(data){
		if (data["success"]) {
			$('#modify_password_dialog').addClass('modified');
			$('#created_task_name').text('('+data['message']+')');
			showMsg('prompt_msg',"<%=rb.getString("ChengGong")%>");
		} else {
			showMsg('error_msg',data["message"]);
			cancelModify();
		}
	}, "json");
}
// 调整到任务列表
function toTaskPage() {
	cancelModify();
	//goMenuPage("70024", false);
	try{
		eventAllBus.$emit("gomenupage","70024","","70024",false);
	}catch(e){}
}
function cancelModify() {
	$('#modify_password_dialog').dialog('close');
}
// 更新已选设备信息
function refreshSelectedEnbList(delay){
	if(isBatchable() !== true) return false;
		
	var tb = $('#tableHomeCpeList'),
		ctn = $(".enb-list-ctn .list-body"),
		rows = tb.datagrid('getChecked');
	ctn.html('');
	rows.map(function(row){
		var item = '<div class="list-item-info">' +row.CPE_NAME + ' (' + row.SERIAL_NUMBER + ') <span class="list-item-op" onclick="clearSingleRecord(\''+row.CPE_CODE+'\',this)">x</span></div>'
		ctn.append(item);
	})
	if(rows.length) {
		showModifySlider();
		$('#device_count').text(rows.length);
	}else {
		var time = 0;
		if(delay === true) time = 500;
		setTimeout(function(){
			showModifySlider(false);
		},time);
	}
}
function delAllSelectedRecord(){
	$("#tableHomeCpeList").datagrid('clearSelections').datagrid('clearChecked');
	$(".enb-list-ctn .list-body").html('');
}
function clearSingleRecord(id,span){
	var tb = $("#tableHomeCpeList"),
		rows = tb.datagrid('getSelections'),
		crows = tb.datagrid('getChecked');
	
	var rIdx = -1;
	rows.map(function(item,idx){
		if(item.CPE_CODE == id) rIdx = idx;
	});
	if(rIdx>=0) {
		rows.splice(rIdx,1);
		crows.splice(rIdx,1);
	}
	var index = tb.datagrid('getRowIndex',id);
	if(index>=0){
		tb.datagrid('unselectRow',index);
		tb.datagrid('uncheckRow',index);
	}
	
	$(span).parents('.list-item-info:first').remove();
}
function showModifySlider(bool) {
	if(bool === false) {
		$('.bottom-slider').removeClass('show');
		showDeviceList(false);
	}else $('.bottom-slider').addClass('show');
}
// 显示已选设备信息层
function showDeviceList(bool) {
	var arrow = $('#sliderArrow');
	if(bool === false) {
		$('.selected-devices-info').removeClass('show');
		arrow.addClass('el-icon-circle-down').removeClass('el-icon-circle-up');
		deviceFlag = false;
	}else {
		if(deviceFlag){
			$('.selected-devices-info').removeClass('show');
			arrow.addClass('el-icon-circle-down').removeClass('el-icon-circle-up');
			deviceFlag = false;
		}else{
			$('.selected-devices-info').addClass('show');
			arrow.addClass('el-icon-circle-up').removeClass('el-icon-circle-down');
			deviceFlag = true;
		}
		
	}
}
//APN复选框点击事件
function apnCheckFun(ele){
	var operateClassName = $(ele).parent()[0].className.split(" ")[0];
	if(!$(ele).is(":checked")){
		$("."+operateClassName).each(function(){
            $($(this)[0]).children(":first-child").prop("disabled","disabled");
        })
        $(ele).prop("disabled","");
        $(ele).prop("cheched","cheched");
	}else{
		$("."+operateClassName).each(function(){
            $($(this)[0]).children(":first-child").prop("disabled","");
        })
        var apnTypeFlag=$("."+operateClassName+".apnType_row").children(":first-child").val();
		if(apnTypeFlag=='001'){
			$("."+operateClassName+".defaultRouter_row").children(":first-child").prop("disabled","disabled");	
		}else{
			$("."+operateClassName+".vlanList_row").children(":first-child").prop("disabled","disabled");			
		}
	}
	$(".apnIp_col input").prop("disabled","disabled");
}

//APN下拉框改变事件
function apnSelectFun(ele){
	var operateClassName = $(ele).parent()[0].className.split(" ")[0];
	if($(ele).val()=='001'){
		$("."+operateClassName+".vlanList_row").children(":first-child").prop("disabled","");
		$("."+operateClassName+".defaultRouter_row").children(":first-child").prop("checked","");
		$("."+operateClassName+".defaultRouter_row").children(":first-child").prop("disabled","disabled");		
		$("."+operateClassName+".apnName_row").children(":first-child").val("");
		$("."+operateClassName+".apnName_row").children(":last-child").css('color','#D2D2D2');
		$("."+operateClassName+".vlanList_row").children(":first-child").val("");
		$("."+operateClassName+".vlanList_row").children(":last-child").css('color','#D2D2D2');
	}else{
		$("."+operateClassName+".apnName_row").children(":first-child").val("");
		$("."+operateClassName+".apnName_row").children(":last-child").css('color','#D2D2D2');
		$("."+operateClassName+".vlanList_row").children(":first-child").val("");
		$("."+operateClassName+".vlanList_row").children(":last-child").css('color','#D2D2D2');
		$("."+operateClassName+".vlanList_row").children(":first-child").prop("disabled","disabled");
		$("."+operateClassName+".defaultRouter_row").children(":first-child").prop("disabled","");
	}
}
// 加载前事件-CPE列表
function tableHomeCpeListBeforeLoad(param) {
	
}
//CPE列表-选择列表之前
function onBeforeSelect(index,row){
	 var tb = $(this);
	slData = tb.datagrid('getChecked'),
	rows = tb.datagrid('getRows');
	$.each(rows,function(index,item){
		if($.inArray(item,slData) < 0 && item!=row) {
			var rowIndex = tb.datagrid('getRowIndex',item);
			tb.datagrid('unselectRow',rowIndex);
		}    
	});
}

// 显示右键-CPE列表
function showRowMenutableHomeCpeList(e, rowIndex, rowData) {
	if(rowData.connection_status == 'Off'){
   	 var itemEl=$("#tongbuCpe")[0];
   	 var itemE2=$("#chongqiCpe")[0];
   	 $("#rowMenuTableHomeCpeList").menu("disableItem",itemEl); 
   	 $("#rowMenuTableHomeCpeList").menu("disableItem",itemE2); 
    }else{
   	 var itemEl=$("#tongbuCpe")[0];
   	 var itemE2=$("#chongqiCpe")[0];
   	 $("#rowMenuTableHomeCpeList").menu("enableItem",itemEl); 
   	 $("#rowMenuTableHomeCpeList").menu("enableItem",itemE2); 
    }
	e.preventDefault();
	if (rowIndex < 0) {
		return;
	}
	if($("#cpeSetting").offset().left<1050){
		return;
	} 
	$("#tableHomeCpeList").datagrid("clearSelections");
    $("#tableHomeCpeList").datagrid("selectRow", rowIndex);
    $("#rowMenuTableHomeCpeList").menu("show", {
        left: e.clientX,
        top: e.clientY
    });
    
    var selCpe = $("#tableHomeCpeList").datagrid("getSelected");
    if(isBatchable() == true && curCPECode ) { // 批量修改密码开关
		var allRows = $("#tableHomeCpeList").datagrid("getRows");
		allRows.map(function(row){
			if(row.CPE_CODE == curCPECode) selCpe = row;
		});
	}
    var product = selCpe["OLDPRODUCT"];
	var reg = new RegExp("^(IDU\/CN)");
	if (product == "LTE WiFi VoIP Gateway" || (reg.test(product)==true)) {
    	$(".easyui-menu #freqLock").attr("onclick","");
    	$(".easyui-menu #freqLock").validatebox({disabled:true});
    	$(".easyui-menu #freqLock").attr("class","menu-item menu-item-disabled validatebox-text");
    } else {
    	$(".easyui-menu #freqLock").validatebox({disabled:false});
    	$(".easyui-menu #freqLock").attr("onclick","cpeFreqLock()");
    	$(".easyui-menu #freqLock").attr("class","menu-item validatebox-text");
    }
}

// 打开CPE详细信息窗口
function goCpeDetailParamInfoWin(divId) {
	 var selCpe = $("#tableHomeCpeList").datagrid("getSelected");
	 
	if(isBatchable() == true && curCPECode ) { // 批量修改密码开关
		var allRows = $("#tableHomeCpeList").datagrid("getRows");
		allRows.map(function(row){
			if(row.CPE_CODE == curCPECode) selCpe = row;
		});
	}
	if (!selCpe) {
		showMsg('prompt_msg','<%=rb.getString("QingXuanZeSheBei")%>');
		return;
	}
	sessionStorage.setItem('oldProduct', selCpe.OLDPRODUCT);
	sessionStorage.setItem('CONNECTION_STATUS',selCpe.CONNECTION_STATUS);
	sessionStorage.setItem('CPE_CODE',selCpe["CPE_CODE"]);
	sessionStorage.setItem('PRODUCT',selCpe["PRODUCT"]);
	sessionStorage.setItem('cpeName',selCpe["CPE_NAME"]);
	$("#cpeInformation").load("${ctx}/cell/CPE/toCpeDetailParamInfoPage.action?cpeCode=" + selCpe["CPE_CODE"]+'&timeZone='+timeZone, function(data){
		$.parser.parse(this);
    });
	judmentOtherChange(divId);
}

// 刷新某个CPE信息
function refreshCpe() {
	 var selCpe = $("#tableHomeCpeList").datagrid("getSelected");
	 
	if(isBatchable() == true && curCPECode ) { // 批量修改密码开关
		var allRows = $("#tableHomeCpeList").datagrid("getRows");
		allRows.map(function(row){
			if(row.CPE_CODE == curCPECode) selCpe = row;
		});
	}
	 var rowIndex=$("#tableHomeCpeList").datagrid("getRowIndex",selCpe.CPE_CODE);
	 selCpe.CONNECTION_STATUS = 'updating';
	if (!selCpe) {
		showMsg('prompt_msg','<%=rb.getString("QingXuanZeSheBei")%>');
		return;
	}
	$("#tableHomeCpeList").datagrid("updateRow",{
		index:rowIndex,
		row:selCpe
	})
	var param = {cpeCode: selCpe["CPE_CODE"]};
	$.post("${ctx}/cell/CPE/refreshCpeInfo.action", param, function(data){
		if (data["success"]) {
			 //$("#tableHomeCpeList").datagrid("load");
		} else {
			showMsg('error_msg',data["message"]);
		}
	}, "json");
}
function exportCpeInfo(){
	
	//问题单号 24051 3. 考虑到性能 暂时禁用admin多选运营商同时导出 admin默认登出default数据
	autoExportCpesToExcel();
}
//导出
function autoExportCpesToExcel() {
	var params = {};
	// 已选运营商
	params["operator_codes"] = operator_code;
	params["sort"] = sortCPE;
	params["order"] = orderCPE;
	// 已选列
	var selContent = "";
	// 原表格参数
	var options = $("#tableHomeCpeList").datagrid("options");
	/** 去掉选择列后新增的列获取逻辑 **/
	var columns = options.columns;
	columns[0].map(function(item){
		if(item.hidden != true){
			if (selContent != "") {
				selContent += ",";
			}
			selContent += item.field;
		}
	});
	/** ------------------------ **/
	if (selContent == "") {
		showMsg('prompt_msg','<%=rb.getString("XuanZeDaoChuNeiRong")%>');
		return;
	}
	params.content = selContent;

	
	// 原表格查询条件
	//$.extend(params, options["queryParams"]);
	if(options["queryParams"]){
		$.each(options["queryParams"],function(key,val){
			var prop = key;
			if(prop && prop.indexOf('qry_map')>=0) prop = prop.replace('qry_map.','');
			params[prop] = val;
		});
	}
	
	params["TimeZone"] = timeZone;
	// 提交表单
	exportByForm($("#formExportCpeList").attr('action'),params);
}
// 打开导出设置窗口
function openWinExportConfig_cpe() {
	exportCpeInfo();
}

<%-- 重启 --%>
function cpeReboot() {
	 var selCell = $("#tableHomeCpeList").datagrid("getSelected");
	 
	if(isBatchable() == true && curCPECode ) { // 批量修改密码开关
		var allRows = $("#tableHomeCpeList").datagrid("getRows");
		allRows.map(function(row){
			if(row.CPE_CODE == curCPECode) selCell = row;
		});
	}
	
	// 单子30936：后端确认直接发起重启请求
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingChongQiSheBei")%>", function (r) {
		if (r) {
			var param = {
				cpe_code: selCell["CPE_CODE"]
			};
			$.post("${ctx}/cell/CPE/cpeReboot.action", param, function (data) {
				if (!data["success"]) {
					showMsg('error_msg',data["message"]);
				}
			}, "json");
		}
	}).addClass('seriousConfirm');
}

function cpeFreqLock() {
	var selCpe = $("#tableHomeCpeList").datagrid("getSelected");
	 
	if(isBatchable() == true && curCPECode ) { // 批量修改密码开关
		var allRows = $("#tableHomeCpeList").datagrid("getRows");
		allRows.map(function(row){
			if(row.CPE_CODE == curCPECode) selCpe = row;
		});
	}
	var url="${ctx}/cell/CPE/toCpeFreqLock.action?cpeCode=" + selCpe["CPE_CODE"],
		options={
			title: "<%=rb.getString("SuoPin")%>",
			width: 430,
			height: 280
		};
	openDefaultWindow(url,options);
}

function setCpeHttpsWanConn() {
	var selCell = $("#tableHomeCpeList").datagrid("getSelected");
	
	if(isBatchable() == true && curCPECode ) { // 批量修改密码开关
		var allRows = $("#tableHomeCpeList").datagrid("getRows");
		allRows.map(function(row){
			if(row.CPE_CODE == curCPECode) selCell = row;
		});
	}
	//判断基站是否连接
	$.post("${ctx}/cell/cpeinfos/getCpeConnStatus.action", {"small_cell_code": selCell["CPE_CODE"]}, function(data){
		// 基站未连接，不允许修改参数
		if(data["connStatus"] == "false") {
			showMsg('prompt_msg','<%=rb.getString("CpeWeiLianJie")%>');
			return;
		} else {
			showMsg('prompt_msg','<%=rb.getString("CanShuXiaFa")%>');
			var param = {
				cpe_code: selCell["CPE_CODE"]
			};
			$.post("${ctx}/cell/CPE/setHttpsWanConn.action", param, function (data) {
			}, "json");
				
		}
	},"json");
}

function showHisRecord(type) {
	/* $("#winOperationResultForCpe").window("center").window("open");
	$("#winOperationResultForCpe").window("refresh", "${ctx}/cell/CPE/toOperationResultForCpe.action"); */
	var url = "${ctx}/cell/CPE/toOperationResultForCpe.action",
		options = {
			title:'<%=rb.getString("LiShi")%>',
			width:1150,
			height:600
		};
	openDefaultWindow(url,options);
}

//根据RSRP值范围，显示
function showRedAccordingRSRP(value, rowData, rowIndex) {
	var maxValue = highVal,
	    minValue = lowVal;
	
	var ret ;
	if ( value ){
		if (value < minValue) {
			ret = "<span class='el-icon el-icon-signal signal-low'>" + value + "</span>";
			
		} else if (value > maxValue){
			ret = "<span class='el-icon el-icon-signal signal-high'>" + value + "</span>";
		}else {
			ret = "<span class='el-icon el-icon-signal signal-normal'>" + value + "</span>";
		}
	}else {
		ret = value || ''
	}
	
	return ret;
}

/*page global var declare  */
var rsrpGood ="${RSRP_1}" ;
var rsrpNormal ="${RSRP_2}" ;
var rsrpBad ="${RSRP_3}" ;
var cinrGood ="${CINR_1}" ;
var cinrNormal ="${CINR_2}" ;
var cinrBad ="${CINR_3}" ; 

function formatRsrp(value, rowData, rowIndex){
	if (!value) return "";
	value = value * 1;
	var retStr = "<span><font color='black'>" + value + "</font></span>";
	retStrGood = "<span ><font color='#4BCC88'>" + value + "</font></span>";
	retStrNormal = "<span><font color='#DDD732'>" + value + "</font></span>";
	retStrBad = "<span><font color='#D3234C'>" + value + "</font></span>";
	var array = rsrpGood.split(",");
	var goodValueMin = array[0];
	var goodValueMax=array[1];
	if("-" == goodValueMin){
		if(value <= Number(goodValueMax)){
			//好
			retStr = retStrGood;
			return retStr;
		}
	} else if("-" == goodValueMax){
		if(value >= Number(goodValueMin)){
			//好
			retStr = retStrGood;
			return retStr;
		}
	} else {
		if(Number(value) >= Number(goodValueMin)){
			if(Number(value) <= Number(goodValueMax)){
				retStr = retStrGood;
				return retStr;
			}
		}
	}
		
	var array1 = rsrpNormal.split(",");
	var normalValueMin = array1[0];
	var normalValueMax=array1[1];
	if("-" == normalValueMin){
		if(value <= Number(normalValueMax)){
			//中
			retStr = retStrNormal;
			return retStr;
		}
	} else if("-" == normalValueMax){
		if(value >= Number(normalValueMin)){
			//中
			retStr = retStrNormal;
			return retStr;
		}
	} else {
		if(value >= Number(normalValueMin) &&value <= Number(normalValueMax)){
			//中
			retStr = retStrNormal;
			return retStr;
		}
	} 
		
	var array2 = rsrpBad.split(",");
	var badValueMin = array2[0];
	var badValueMax=array2[1];
	if("-" == badValueMin){
		if(value <= Number(badValueMax)){
			//差
			retStr = retStrBad;
			return retStr;
		}
	} else if("-" == badValueMax){
		if(value >= Number(badValueMin)){
			//差
			retStr = retStrBad;
			return retStr;
		}
	} else {
		if(value >= Number(badValueMin)){
			if(value <= Number(badValueMax)){
				//差
				retStr = retStrBad;
				return retStr;
			}
		}
	} 
 	return retStr;
}

function formatCinr(value, rowData, rowIndex){
	if (!value) return "";
	var retStr = "<span><font color='black'>" + value + "</font></span>";
	retStrGood = "<span ><font color='#4BCC88'>" + value + "</font></span>";
	retStrNormal = "<span><font color='#DDD732'>" + value + "</font></span>";
	retStrBad = "<span><font color='#D3234C'>" + value + "</font></span>";
	var array = cinrGood.split(",");
	var goodValueMin = array[0];
	var goodValueMax=array[1];
	if("-" == goodValueMin){
		if(value <= Number(goodValueMax)){
			//好
			retStr = retStrGood;
			return retStr;
		}
	} else if("-" == goodValueMax){
		if(value >= Number(goodValueMin)){
			//好
			retStr = retStrGood;
			return retStr;
		}
	} else {
		if(value >= Number(goodValueMin) && value <= Number(goodValueMax)){
			//极好
			retStr = retStrGood;
			return retStr;
		}
	}
		
	var array1 = cinrNormal.split(",");
	var normalValueMin = array1[0];
	var normalValueMax=array1[1];
	if("-" == normalValueMin){
		if(value <= Number(normalValueMax)){
			//好
			retStr = retStrNormal;
			return retStr;
		}
	} else if("-"== normalValueMax){
		if(value >= Number(normalValueMin)){
			//好
			retStr = retStrNormal;
			return retStr;
		}
	} else {
		if(value >= Number(normalValueMin)){
			if(value <= Number(normalValueMax)){
				//极好
				retStr = retStrNormal;
				return retStr;
			}
		}
	}
		
	var array2 = cinrBad.split(",");
	var badValueMin = array2[0];
	var badValueMax=array2[1];
	if("-" == badValueMin){
		if(value <= Number(badValueMax)){
			//好
			retStr = retStrBad;
			return retStr;
		}
	} else if("-" == badValueMax){
		if(value >= Number(badValueMin)){
			//好
			retStr = retStrBad;
			return retStr;
		}
	} else {
		if(value >= Number(badValueMin) && value <= Number(badValueMax)){
			//极好
			retStr = retStrBad;
			return retStr;
		}
	}
	
	return retStr;
}

function ColomnStatus(divId){
	$('#'+divId).slideDown();
	var boxes = document.getElementsByName("contentM");
	for(i=0;i<boxes.length;i++){
		boxes[i].checked = true;
	}
 	if("" != columncpe){
		var strs = new Array();
		strs = columncpe.split(",");
		for(i=0;i<boxes.length;i++){
			for(j=0;j<strs.length;j++){
				if(boxes[i].value == strs[j]){
					boxes[i].checked = false;
					break;
				}
			}
		}
		document.getElementsByName("all_content2")[0].checked = false;
	} 
}

function cpeMonitorDatagridLoadSuccess(){
	$(this).datagrid("enableContextmenuAutoSize");
	refresh_cpeStatusStatistics();
	if("" != columncpe){
		var strs = new Array();
		strs = columncpe.split(",");
		for(i=0;i<strs.length;i++){
			/* $("#tableHomeCpeList").datagrid("hideColumn",strs[i]); */
		}
	}
	$("#tableHomeCpeList").prev().find(".datagrid-body td[field='HISTORY_DATA']").addClass("cpeDataHis");	
	$("#tableHomeCpeList").prev().find(".datagrid-body td[field='UL_MCS']").addClass("cpeData");
	$("#tableHomeCpeList").prev().find(".datagrid-body td[field='DL_MCS']").addClass("cpeData");
	$("#tableHomeCpeList").prev().find(".datagrid-body td[field='RSRP0']").addClass("cpeData");
	$("#tableHomeCpeList").prev().find(".datagrid-body td[field='RSRP1']").addClass("cpeData");
	$("#tableHomeCpeList").prev().find(".datagrid-body td[field='CINR0']").addClass("cpeData");
	$("#tableHomeCpeList").prev().find(".datagrid-body td[field='CINR1']").addClass("cpeData");
	$("#tableHomeCpeList").prev().find(".datagrid-body td[field='CPE_SINR']").addClass("cpeData");
	$("#tableHomeCpeList").prev().find(".datagrid-body td[field='DL_CURRENT_DATARATE']").addClass("cpeData");
	$("#tableHomeCpeList").prev().find(".datagrid-body td[field='UL_CURRENT_DATARATE']").addClass("cpeData");
	$(this).datagrid("fixRownumber");
}

//导出内容，全选框，点击事件
function allCkOnClick_cpeM(event) {
	var checked = event.target.checked;
	if (checked) {
		$(".cpe1ExportConfigItem input[type='checkbox']:not('.cpeunCheckedBox')").each(function() {
			this.checked = true;
		});
	} else {
		$(".cpe1ExportConfigItem input[type='checkbox']:not('.cpeunCheckedBox')").each(function() {
			this.checked = false;
		});
	} 
}

function close1ColumnConfigCpe(){
	var boxes = document.getElementsByName("contentM");
	for(i=0;i<boxes.length;i++){
		boxes[i].checked = true;
	}
	if("" != columncpe){
		var strs = new Array();
		strs = columncpe.split(",");
		for(i=0;i<boxes.length;i++){
			for(j=0;j<strs.length;j++){
				if(boxes[i].value == strs[j]){
					boxes[i].checked = false;
					break;
				}
			}
		}
		document.getElementsByName("all_content2")[0].checked = false;
	}
	var allcheckLength = $(".cpe1ExportConfigItem").length;
	var checkedboxLength = $("input[type='checkbox'][name = 'contentM']:checked ").length; 
	if(allcheckLength == checkedboxLength){
		$("#all_contentN_cpeExportConfig").prop("checked",true);
	} 
	
	$(".showHideItem").slideUp(500); 
	if(allColumn != ""){
		var dataGridItem = $(".cpe1ExportConfigItem").clone();
		$(".sortul li").remove();
	    var fakeDataArr = itemList.split(",");
		 for(var i=0;i<fakeDataArr.length;i++){
			for(var j=0;j<dataGridItem.length;j++){
				if(fakeDataArr[i] == ($(dataGridItem[j]).find("input").attr("item"))){
					$(".sortul").append(dataGridItem[j]);
				}
			}
		}
		 $(".sortul li").first().find(".el-icon-moveTop").addClass("disabled");
	}else{
		$(".sortul li").first().find(".el-icon-moveTop").addClass("disabled");
	};
	
	$(".sortul .el-icon-moveTop:not(:first)").removeClass("disabled");
	//重新绑定事件
	$(".sortul input").click(function(){
		if(!$(".sortul input").checked){
			$("#all_contentN_cpeExportConfig").attr("checked",false);
		}
		var allcheckLength = $(".cpe1ExportConfigItem").length;
		var checkedboxLength = $("input[type='checkbox'][name = 'contentM']:checked ").length; 
		if(allcheckLength == checkedboxLength){
			$("#all_contentN_cpeExportConfig").prop("checked",true);
		}
	})
	
	$(".cpe1ExportConfigItem").mousedown(function(event){
		if(event.target.tagName == "INPUT" || event.target.tagName == "LABEL" || (event.target.tagName == "SPAN" && $(event.target).hasClass('el-icon-moveTop'))){
			
		}else{
			$(this).addClass("handleMouseDown");
		}
		
	});
	$(".cpe1ExportConfigItem").mouseup(function(){
		$(this).removeClass("handleMouseDown");
	});
	
	$(".el-icon-moveTop").click(function(){
		var parIndex = $(this).parents("li").index();
		var parentBox = $(this).parents("li");
		if(parIndex != 0){
			$(".sortul").prepend(parentBox);
			$(this).addClass("disabled");
			parentBox.siblings().find(".el-icon-moveTop").removeClass("disabled");
		}
	});
	
}

function Column1ConfigCpe() {
	//itemList 传给后台用来加载排序选项的参数
	itemList = "";
	var checkList = $(".cpe1ExportConfigItem").find("input");
	$(".cpe1ExportConfigItem input[type='checkbox']").each(function() {
		itemList += ($(this).attr("item")+",");	
	});
	itemList = itemList.substring(0,itemList.length-1);
	selContent = "";
	$(".cpe1ExportConfigItem input[type='checkbox']").each(function() {
		if (this.checked) {
			
		}else{
			if (selContent != "") {
				selContent += ",";
			}
			selContent += $(this).attr("item");
		}
	});
	//要传递的参数
	sortcolumn = "";
	//选中的选项作为传递的参数
	$(".cpe1ExportConfigItem input[type='checkbox']").each(function() {
		if (this.checked) {
			if (sortcolumn != "") {
				sortcolumn += ",";
			}
			sortcolumn += $(this).attr("item");
		}
	});
	
	columncpe = selContent;
	allColumn = itemList;
	var params = {
		"hiddenColumn": selContent,
		"allColumn": itemList
	};
	
	$.post("${ctx}/cell/CPE/cpeColumnConfig.action", params, function (data) {
		/* $('#win1ColumnConfig').window('close'); */
	    $(".showHideItem").slideUp(500); 
		if (!data["success"]) {
			showMsg('error_msg','<%=rb.getString("BaoCunShiBai")%>');
		}else{
			var field_arr = [];
			if("" != selContent){
				field_arr = selContent.split(",");
			}
			
			//调整顺序后获取的数组（排序）
		 	var sortColumnArr = sortcolumn.split(",");
			
			var loadtable =$.extend(true,[],cpe_column);
			loadtable.splice(0,loadtable.length);
			if(isBatchable()) {
				loadtable.push({field:'ck',checkbox: true});
			}
			loadtable.push({field:'cpe_operation',fixed: true,width: 30,sortable:false,formatter:cpeTaskFormatter,styler:setStyle,title:''});
			loadtable.push({field:'CONNECTION_STATUS',title:titleFilter,width:60,sortable:true,formatter:connStatusFormatterSyn});
			loadtable.push({field:'SERIAL_NUMBER',sortable:true,width:170,title:'<%=rb.getString("CPEXuLieHao")%>'});
			for(var i=0;i<sortColumnArr.length;i++){
				for(var j=0;j<cpe_column.length;j++){
					if(cpe_column[j].field == sortColumnArr[i]){
						loadtable.push(cpe_column[j]);
						break;
					}
				}
			}  
			
			var fitColumn = true;
		    var main_width = $("#mainpage").width();
			if((main_width<1280 && loadtable.length<=5 )|| (main_width>=1280 && loadtable.length<=14)){
				fitColumn = true;
			}
			var tmpCols = [];
			$.each(loadtable,function(index,item){
				var col = $.extend({},item);
				tmpCols.push(col);
			});
			(function(fitCol,tmpCols){
				$('#tableHomeCpeList').datagrid({fitColumns:fitCol,columns:[tmpCols]});
				initCPEFilterStatus();
			})(fitColumn,tmpCols); 
			
		}
	}, "json");	
}

function formatCPEType(value, rowData, rowIndex) {
	var reg = new RegExp("^IDU");
	if (value == "LTE WiFi VoIP Gateway" || (reg.test(value)==true)) {
		return "IDU";
	} else {
		return "ODU";
	}
}
/**
 * 数据表格中，列的格式化函数，作用:判断PCI状态，是否锁定
 * @param value 字段的值
 * @param rowData 行的数据
 * @param rowIndex 行的索引
 */
function tablePciStatus(value, rowData, rowIndex){
	var reg = new RegExp("^(IDU\/CN)");
	if("LTE WiFi VoIP Gateway" == rowData["OLDPRODUCT"] || (reg.test(rowData["OLDPRODUCT"])==true) || value == '--'){
		return "--";
	}
	if(!value) return "";
	var pciValue = rowData.PCI;
	if(rowData.SCANMODE == 'pcilock' || rowData.SCANMODE == 'pcionlylock' ){
		var imgL = "<div class='pciClass'><span  title='<%=rb.getString("JieChuSuoDingToolTip")%>' class='el-icon el-icon-operation-lock easyui-tooltip' onclick=pciLockClick('" + rowData.PCI +"','" + rowData.SCANMODE +"','"+ rowData.CPE_CODE +"','" + rowData.PCI + "') ></span></div><span style='margin-left:5px;'>"+pciValue+"</span>" ;
		return imgL;
	}else{
		// if( (rowData.PCI_ONLY_LOCK !== null && rowData.PCI_ONLY_LOCK.trim().length == 0) || rowData.PCI_ONLY_LOCK == undefined || rowData.PCI_ONLY_LOCK == null || rowData.PCI_ONLY_LOCK == 's'|| rowData.PCI_ONLY_LOCK == ''){
		// 	var imgL = "<span title='<%=rb.getString("BuKeSuoDingToolTip")%>' style='font-size:14px'  class='el-icon el-icon-status-unlock easyui-tooltip'></span><span>"+pciValue+"</span>" ;
		// 	return imgL;
		// }else{
		// 	var imgL = "<span title='<%=rb.getString("BangDingToolTip")%>'  style='font-size:14px' class='el-icon el-icon-status-unlock easyui-tooltip' onclick=pciLockClick('" + rowData.PCI_ONLY_LOCK +"','" + rowData.SCANMODE +"','"+ rowData.CPE_CODE + "') ></span><span>"+pciValue+"</span>" ;
		// 	return imgL;
		// }
		var imgL = "<div class='pciClass'><span title='<%=rb.getString("BangDingToolTip")%>' class='el-icon el-icon-status-unlock easyui-tooltip' onclick=pciLockClick('" + rowData.PCI +"','" + rowData.SCANMODE +"','"+ rowData.CPE_CODE+"','" + rowData.PCI + "') ></span></div><span>"+pciValue+"</span>" ;
		return imgL;
		
	}
	
}
function pciLockClick(pciOnlyLock,scanMode,cpeCode,pciValue){
	var tipText = "";
	var params = {};
	params.cpeCode = cpeCode;
	params.timeZone = timeZone;
	if(scanMode == 'pcilock' || scanMode == 'pcionlylock' ){
		params.scanMode='fullband';
		tipText = '<%=rb.getString("JieChuSuoDingTanChuangTiShi")%>'; 
	}else{
		params.PCI_value = pciOnlyLock;
		params.scanMode='pcionlylock';
		tipText = '<%=rb.getString("BangDingTanChuangTiShiOne")%>'+ pciValue + '<%=rb.getString("BangDingTanChuangTiShiTwo")%>'; 
	}

	$.messager.confirm('<%=rb.getString("QueRen")%>', tipText, function (r) {
			if (r) {
				$.post("${ctx}/cell/CPE/setCpeParams.action", params, function(data) {
					if (data["success"]) {
						showMsg('prompt_msg','<%=rb.getString("SuoPingRenWuJianLiTiShi")%>');
						$("#tableHomeCpeList").datagrid("reload");
					} else {
						showMsg('error_msg',data["message"]);
						return;
					}
				}, "json");
			}
		}).addClass('seriousConfirm');
}
function historyGraph(value, rowData, rowIndex){
	var imgL = "<span class='el-icon el-icon-operation-statistics' onclick=showHistoryGraph('" + rowData.CPE_CODE + "','" + rowData.SERIAL_NUMBER + "','CPEHistoryCharts','" + rowData.PRODUCT + "')>" + "</span>" ;
	return imgL;
}

//显示CPE历史数据图表
function showHistoryGraph(cpeCode, sn,divId,product) {
    $("#CPEHistoryCharts").load("${ctx}/cell/CPE/toCPEHistoryGraphPage.action", { cpeCode : cpeCode, sn : sn , product : product}, function() {
                //$("#CPEHistoryCharts").animate({ right : '0px' }, 500);
    			//slideOtherDiv(divId);
    			judmentOtherChange(divId);
            });
}

//关闭CPE历史数据图表 
function closeCPEHistory(){
	$("#CPEHistoryCharts").animate({right:'-1300px'},500);
}

//刷新Cpe监控 下 统计信息，填充状态栏
function refresh_cpeStatusStatistics(cb) {
	var params = {};
	var options = $("#tableHomeCpeList").datagrid("options");
	$.extend(params, options["queryParams"]);

	$.post("${ctx}/cell/CPE/getCpeStatusStatistics.action", params, function(data) {
		if(!data["connection_status"]){
			connectionStatus = "0/0";
			connectionStautsRef = "0/0";
			$("#onlineStatusNumCpe").text("0/0");
			//$("#cpeMonitorId .connStatusStatistics").text("0/0");
		}else{
			connectionStatus = data["connection_status"];
			connectionStatusRef = data["connection_status_ref"];
			if($("#onlineStatusNumCpe").prev().hasClass('greenType')){
				$("#onlineStatusNumCpe").text(data["connection_status"]);
			}else if($("#onlineStatusNumCpe").prev().hasClass('redType')){
				$("#onlineStatusNumCpe").text(data["connection_status_ref"]);
			}
			//$("#cpeMonitorId .connStatusStatistics").text(data["connection_status"]);
		}
		$('#cpe_online_count_rate').text('( '+connectionStatus+' )');
		if(cb && typeof cb == 'function') {
			cb();
		}
	}, "json");
}
/*设置提示进度条*/
function progressDivShow(){
	$('.setting').css('display','block');
	$('.set').css('display','none');
	$('#winSettingProCpe').show();
}

/*完成后的进度条*/
function progressDivHide(){	
	$('.setting').hide();
	if(showProcessSetFlag){
		$('.set').hide();
	}else{
		$('.set').fadeIn();
	}
	setTimeout(function(){			
		//$('.newWindow').fadeOut(400);
		$('#winSettingProCpe').fadeOut(400);
	},2000)	
}

function setCpeHttpsWanConn(flag) {
	
	var selCell =  $("#tableHomeCpeList").datagrid("getSelected");	 
	
	if(isBatchable() && curCPECode ) { // 批量修改密码开关
		var allRows = $("#tableHomeCpeList").datagrid("getRows");
		allRows.map(function(row){
			if(row.CPE_CODE == curCPECode) selCell = row;
		});
	}
	var param = {
		cpeCode: selCell["CPE_CODE"],
		enable_flag: flag
	};
	$.post("${ctx}/cell/CPE/setHttpsWanConn.action", param, function (data) {
		
		if (!data["success"]) {
			showMsg('error_msg',data["message"]);
		}
	}, "json");
}

function IpChangeRed(ele){
	$(ele).keyup(function(){
		var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/ ;
	    if (reg.test($(ele).val())) {
	    	$(ele).css('border-color','initial');
	        $($(ele).parent().next()).css('color','#D2D2D2');
	    } else {
	    	$(ele).css('border-color','initial');
	        $($(ele).parent().next()).css('color','#D2D2D2');
	    	var vip = $("#vxlanserverip").val();
	    	if((vip.replace(/\s/g,"")).length < 1){
	    		$(ele).css('border-color','#E64242');
	    		$($(ele).parent().next()).css('color','#E64242');
	    	}
		} 
	})
}
function idChangeRed(ele){
	$(ele).keyup(function(){
		var reg = /^(\d{1,24})$/;
	    if (reg.test($(ele).val())) {
	    	$(ele).css('border-color','initial');
	        $($(ele).parent().next()).css('color','#D2D2D2');
	    } else {
	        $(ele).css('border-color','#E64242');
	        $($(ele).parent().next()).css('color','#E64242');
		} 
	})
}

/* table弹出框的X点击函数 */
function closeCpeSetting(){
	var settingInputText = $("#cpeSetting input[type=text]");
	var settingInputCheckbox = $("#cpeSetting input[type=checkbox]");
	var settingselect = $("#cpeSetting select");
	$.each(settingInputText,function(index,ele){
		$(ele).val($(ele).attr('oldvalue'))  ;
	})
	$.each(settingInputCheckbox,function(index,ele){
		$(ele).attr('value',$(ele).attr('oldvalue'))  ;
	})
	$.each(settingselect,function(index,ele){
		$(ele).val($(ele).attr('oldvalue'));
	})
    //compareCancel();
    $("#cpeSetting").animate({right:'-1700px'},500);
    $("#qosOptionDetails input").css('border-color','');
    $("#qosOptionDetails .inputTipCss").css('color','#D2D2D2');
}	
function editCpe(divId,cpe_code,product,softVersion,cpe_name,lanFlag){
	var selCpe = $("#tableHomeCpeList").datagrid("getSelected");
	 
	if(isBatchable() && curCPECode ) { // 批量修改密码开关
		var allRows = $("#tableHomeCpeList").datagrid("getRows");
		allRows.map(function(row){
			if(row.CPE_CODE == curCPECode) selCpe = row;
		});
	}
	if (!selCpe) {
		showMsg('prompt_msg','<%=rb.getString("QingXuanZeSheBei")%>');
		return;
	}
	sessionStorage.setItem('lanFlag',lanFlag);
	sessionStorage.setItem('oldProduct', selCpe.OLDPRODUCT);
	sessionStorage.setItem('cpeSN', selCpe.SERIAL_NUMBER);
	sessionStorage.setItem('softVersion', softVersion);
	sessionStorage.setItem('cpeName', cpe_name);
	var url = "${ctx}/cell/CPE/toCpeSettingPage.action";
	$("#cpeSettingOption").load(
			url,{cpeCode : cpe_code,product : product,softVersion : softVersion,cpeName : cpe_name},
			function(data){
			$.parser.parse(this);
    });
	judmentOtherChange(divId);
}
function apnSet(divId,cpe_code){
	$("#apnSetting").load(
			"${ctx}/cell/CPE/toCpeAPNDetailParamInfoPage.action?cpeCode=" + cpe_code,
			function(data){
			$.parser.parse(this);
    });
	judmentOtherChange(divId);
}
function cpeNameChangeRed(ele){
	$(ele).keyup(function(){		
		var currValLength = $(ele).val().length;
	    var maxLength = $(ele).attr("max_length");
	    if (maxLength) {// 大于最大值
	    	if (currValLength > maxLength) {
	    		$(ele).next().css('color','#E64242');	    	
	    	}else{
	    		$(ele).next().css('color','#D2D2D2');	    	
	    	}
	    }
	})
}

function closeInfoWindow(){
	$("#cpeInformation").animate({right:'-1000px'},600);
}	
var slideDivArr = [
    {
    	divId:'cpeInformation',
 		position :'right',
    	value:'-1000px'
    },
    {
    	divId:'cpeSettingOption',
 		position :'right',
 		value:'-1700px'
    },
    {
    	divId:'CPEHistoryCharts',
 		position :'right',
 		value:'-1300px'
    },
    {
    	divId:'showHideItemCpe',
    	position :'slideDown'
    },
    {
    	divId:'apnSetting',
    	position:'right',
    	value:'-1700px'
    }
];

/**
 * 格式化操作
 */
function cpeTaskFormatter(value, rowData, rowIndex){
	var rowDatas = rowData;
	var rowIndexs = rowIndex;
	var rowstr = unicodeSpace(JSON.stringify(rowData));
	value = "<div class='el-icon el-icon-operation-more' title='"+ CaoZuo+"' cpecode='"+rowDatas.CPE_CODE+"' onclick='choseCpeOp("+ rowstr +",this)'></div>";
	return value;
}  
/**
 * 格式化操作 capablity
 */
function capablityFormatter(value,rowData,rowIndex){
	var capablity = rowData.CAPABILITY;
	var lte_turbo_enalbe = rowData.LTE_TURBO_ENABLE;
	var rowIndexs = rowIndex;
	if(capablity == '0' && isLWAEnable){
		value = "<span class='cpeLwaWu' style='font-size:22px'>"+ "<span style='font-size:12px;margin-left:25px'>"+(value)+"</span>"+"</span>"
	}else if(capablity =='1' && lte_turbo_enalbe == '1' && isLWAEnable){
		value = "<span class='cpeLwaKai' style='font-size:22px'>"+ "<span style='font-size:12px;margin-left:25px'>"+(value)+"</span>"+"</span>"
	}else if(capablity == '1' && lte_turbo_enalbe == '0' && isLWAEnable){
		value = "<span class='cpeLwaGuan' style='font-size:22px'>"+ "<span style='font-size:12px;margin-left:25px'>"+(value)+"</span>"+"</span>"
	}
	return value;
}
/**
* 针对单引号和空格转换 -- 加密
* @param str{string}: 要加密的字符
**/
function unicodeSpace(str){
	return str.replace(/\s/g,'@space@').replace(/'/g,'@dot@');
}
/**
* 针对单引号和空格转换 -- 解密
* @param str{string}: 要解密的字符
**/
function decodeSpace(str){
	return str.replace(/@space@/g,' ').replace(/@dot@/g,"'");
}
//点击行内【更多】按钮，下拉显示操作选项
function choseCpeOp(data,e){
	var cpeRowData = JSON.parse(decodeSpace(JSON.stringify(data)));
	var status = cpeRowData.CONNECTION_STATUS;
	var oldproduct = cpeRowData.OLDPRODUCT;
	var softversion = cpeRowData.SOFTWARE_VERSION;
	var cpe_code = cpeRowData.CPE_CODE;
	var cpe_name = cpeRowData.CPE_NAME;
	var capablity = cpeRowData.CAPABILITY;
	var lte_turbo_enalbe = cpeRowData.LTE_TURBO_ENABLE;
	var lanInterface = cpeRowData.LAN_INTERFACE;
	var connectStatus = status;
	var XinXi = '<%=rb.getString("XinXi")%>';
	var TongBu = '<%=rb.getString("TongBu")%>';
	var SheZhi = '<%=rb.getString("SheZhi")%>';
	var ChongQi = '<%=rb.getString("ChongQi")%>';
	var SuoPin = '<%=rb.getString("SuoPin")%>';
	var GengDuoCaoZuo = '<%=rb.getString("GengDuoCaoZuo")%>';
	var apnSheZhi = 'APN/L2 <%=rb.getString("SheZhi")%>'
	var Kai =  'LTE-TURBO <%= rb.getString("KeYong")%>';
	var Guan = 'LTE-TURBO <%= rb.getString("BuKeYong")%>';
	
	if(isBatchable()) { // 批量修改密码开关
		curCPECode = $(e).attr('cpecode');
		var idx = $("#tableHomeCpeList").datagrid('getRowIndex');
		if(idx>=0) $("#tableHomeCpeList").datagrid('selectRow',idx);
	}
	//同步
	var tongbuDisableFlag;
	if(connectStatus != 'Off'){
		tongbuDisableFlag = false;
	}else{
		tongbuDisableFlag = true;
	}
	//锁频是否可用
	 var suopinDisableFlag;
	var product = oldproduct;
	
	//重启
	var chongQidisableFlag;
	chongQiShowFlag = true;
	if(connectStatus != 'Off'){
		chongQidisableFlag = false;
	}else{
		chongQidisableFlag = true;
	}
	if(cpe_name == 'null'){
		cpe_name = '';
	}
	/* apn设置是否禁用 */
	var apnDisableFlag = false;
	var reg = new RegExp("^(IDU\/CN)");
	var regIduEG = new RegExp("^(IDU\/EG)");
	var regOduEG = new RegExp("^(ODU\/EG)");
	var regu4G = new RegExp("^((ODU\/u4G)|(IDU\/u4G))");
	var l2flag = false;
	if (product == "LTE WiFi VoIP Gateway" || (reg.test(product)==true)) {
		l2flag = false;
    } else if(regIduEG.test(product)){
		l2flag = true;
	} else if(regOduEG.test(product)){
		l2flag = true;
	} else if(regu4G.test(product)){
		l2flag = true;
	} else {
		l2flag = false;
	}
	if(l2flag){
		apnDisableFlag = false;
	} else {
		apnDisableFlag = true;
	}
	var showKai = false ;
	var showGuan = false;
	var notUserFlag = false;
	var capablity = capablity;
	var lte_turbo_enalbe = lte_turbo_enalbe;
	if(lte_turbo_enalbe == '0' && capablity!='0'){ // turbo为关闭状态 显示开按钮
		showKai = true
	}
	if(lte_turbo_enalbe == '1' && capablity!='0'){ // turbo为开启状态 显示关按钮
		showGuan = true
	}
	if(!isLWAEnable) {
		showGuan = false;
		showKai = false;
	}
	var data = [
		{text:XinXi,id:"cp1",cls:'el-icon el-icon-operation-info',show:true},
		{text:SheZhi,id:"cp2",cls:'el-icon el-icon-operation-settings CODE_CPE_SETTINGS hidden',show:true,cpe_code:cpe_code,product:oldproduct,softVersion:softversion,cpe_name:cpe_name,LAN_INTERFACE: lanInterface},
		{text:GengDuoCaoZuo,id:'cp',cls:'el-icon el-icon-operation-more-circle CODE_CPE_SYNCHRONIZE CODE_CPE_REBOOT hidden', //CODE_CPE_APN
        	children: [
        		{text:TongBu,id:"cp3",cls:'CODE_CPE_SYNCHRONIZE hidden',show:false,disable:tongbuDisableFlag},
        		{text:ChongQi,id:"cp5",cls:'CODE_CPE_REBOOT hidden',show:chongQiShowFlag,disable: chongQidisableFlag},
        		// {text:apnSheZhi,id:"cp6",cls:'CODE_CPE_APN hidden',cpe_code:cpe_code,disable:apnDisableFlag}
        	]
        },
		{text:Guan,id:'cp7',cls:'el-icon el-icon-operation-disable1 CODE_CPE_SETTINGS hidden',show:showGuan,cpe_code:cpe_code},
	    {text:Kai,id:'cp8',cls:'el-icon el-icon-operation-enable1 CODE_CPE_SETTINGS hidden',show:showKai,cpe_code:cpe_code},
	];
	showcpeMenu(data);
	var thisTop = $(e).offset().top;
	var allHeight = $(document).height();
	if((allHeight - thisTop) <280){
		$('#cpemn').css({
			"top":thisTop-160,
			"left": 75
		});
	}else{
		$('#cpemn').css({
			"top":thisTop - 20,
			"left": 75
		});
	}
	
	$('#cpemn').show();
}
function showcpeMenu(data){
	$('#cpemn').cmenu({data:data,click:clickcpeEvent});           
}
function clickcpeEvent(row){
    switch(row.id){
    case "cp1": //信息
    	goCpeDetailParamInfoWin('cpeInformation');
    	break;
    case "cp2"://设置
    	editCpe('cpeSettingOption',row.cpe_code,row.product,row.softVersion,row.cpe_name,row.LAN_INTERFACE);
    	break;
    case "cp3"://同步
    	refreshCpe();
    	break;
    case"cp4"://锁频
    	cpeFreqLock();
   		break;
    case "cp5"://重启
    	cpeReboot();
	   	break;
    case "cp6":
    	apnSet('apnSetting',row.cpe_code);
    	break;
	case "cp7": //  关
    	operLTE('0',row.cpe_code);
    	break;
	case "cp8": // 开
    	operLTE('1',row.cpe_code);
    	break;
	}
	
    $('#cpemn').hide();
}
//设置操作列单元格样式 
function setStyle(){
	return 'position:relative';
}
// 开关
function operLTE(status,cellCode){
	//1-开启 0-关闭
	let Msg = ''
	if(status==="0"){
		Msg = "<%=rb.getString("PiLiangGuanCapacity")%>"
	}else{
		Msg = "<%=rb.getString("PiLiangKaiCapacity")%>"
	}
	$.messager.confirm('<%=rb.getString("QueRen")%>', Msg, function (r) {
			if (r) {
				var params = {
					cpeCodes : cellCode,
					turboEnable : status
				}
				$.post("${ctx}/cell/ap/turboSetting.action",params,function(data){
					if(data["success"]){
						showMsg('success_msg','<%=rb.getString("ChengGong")%>');
						$("#tableHomeCpeList").datagrid("reload");
					}else{
						showMsg('error_msg',data["message"]);
					}
				},"json")
			}
		}).addClass('normalConfirm');
}
//高级查询
function cpemoreQuerySlideFun(){
	judmentOtherChange("cpeQueryDiv");
	
	if($("#cpeMoreQueryImg").attr("flag")=="1"){
		$("#cpeQueryDiv").slideDown(500);
		$("#cpeMoreQueryImg").attr("flag","0");
		$("#cpeMoreQueryImg").addClass('expanded');
	}else{
		$("#cpeQueryDiv").slideUp(400);
		$("#cpeMoreQueryImg").attr("flag","1");
		$("#cpeMoreQueryImg").removeClass('expanded');
	}	
}
//根据筛选列显示高级查询筛选
function queryCPETable(e){
	var queryParamObj = {}; 
	if(e){
		//先清空高级查询项
		cpeResetQueryInput();
		queryParamObj = $('#cpequeryform').serializeJson();
	}else{
		queryParamObj = $('#cpequeryform').serializeJson();
		queryParamObj['search_text'] = "";
		$("#cpeSearchText").val(null);
	}
    /* var queryParamObj = $('#cpequeryform').serializeJson(); */
    queryParamObj.isCloudCore = isCloudCore;
    queryParamObj.TimeZone = timeZone;
    queryParamObj['like_fields'] =  'HOST_NAME,cpe_name,macaddress,imsi,serial_number';
    globalQueryParams = $.extend({},queryParamObj);
    doSearchUrl('tableHomeCpeList', queryParamObj, '${ctx}/cell/CPE/queryCpeInfosList.action?type=0');
    $("#cpeQueryDiv").slideUp(300);
	$('.highQueryArrow .el-icon').removeClass('el-icon-common-query-up').addClass('el-icon-common-query-down').attr('flag',1)
}
//高级查询重置
function cpeResetQueryInput(){
	/* $("#cpeSearchText").val(null); */
	$("#cpeSN").val(null);
	$("#cpeName").val(null);
	$("#cellName").val(null);
	$("#cpeimsi").val(null);
	$("#cpeipAddr").val(null);
	$("#cpemacaddr").val(null);
	$("#cpecellid").val(null);
	$("#pciInput").val(null);
	$("#cpeLGWIP").val(null);
	$("#cpeLGWMAC").val(null);
	$("#CPESoftVersion").combobox('setValue','');
	$("#CPESoftVersion").combobox('setText',['<%=rb.getString("QuanBu")%>']);
	$("#CPEDeviceGroup").combobox('setValue','');
	$("#CPEDeviceGroup").combobox('setText',['<%=rb.getString("QuanBu")%>']);
	$("#CPEconnStatus").combobox('setValue','');
	$("#CPEconnStatus").combobox('setText',['<%=rb.getString("QuanBu")%>']);
}

function setFilterStyle(tb){
	var $tb = $(tb),
		params = $tb.datagrid('options').queryParams,
		codes = {
			'connection_status':'CONNECTION_STATUS',
			'software_version':'SOFTWARE_VERSION',
			'group_id':'group_name'
		};
	$.each(params,function(key,val){
		if(codes[key]) {
			var span = $('.datagrid-header-row td[field="'+codes[key]+'"] .filter-opt',$tb.parent());
			if(val) span.addClass('selected');
			else span.removeClass('selected');
		}
	});
}
function showOnline(){
	if($(".onlineMenu").is(":visible")){
		$(".onlineMenu").hide();
	}else{
		$(".onlineMenu").show();
	}
}
function onlineClick(type){
	if(type == 'online'){
		$("#onlineStatusNumCpe").prev().addClass("greenType").removeClass("redType");
		$(".connectionName").text('<%=rb.getString("ShouYe_ZaiXian")%>');
		$("#onlineStatusNumCpe").text(connectionStatus);
	}else if(type == 'offline'){
		$("#onlineStatusNumCpe").prev().removeClass("greenType").addClass("redType");
		$(".connectionName").text('<%=rb.getString("ShouYe_BuZaiXian")%>');
		$("#onlineStatusNumCpe").text(connectionStatusRef);
	}
	$(".onlineMenu").hide();
}
function initCPEFilterStatus() {
	$.post(filterUrl,globalQueryParams,function(json){
		var tb = $('#tableHomeCpeList'), 
			map = json[0],
			queryParams = $(tb).datagrid('options').queryParams,
			viewCtn = $(tb).parent('.datagrid-view'),
			headRow = $('.datagrid-view2 .datagrid-header-row', viewCtn);
			
		for(var key in map) {
			var field = key.toUpperCase();
			var td = $('td[field='+field+']', headRow),
				queryStr = queryParams[key],
				filtor = $('.filter-opt', td),
				menus = map[key];
			
			if(key == 'group') {
				td = $('td[field=group_name]', headRow),
				queryStr = queryParams['group_id'],
				filtor = $('.filter-opt', td)
			}
			
			if(queryStr) {
				var list = queryStr.split(',');
				if(menus.length>list.length) {
					filtor.addClass('selected');
				}
			}
		}
	},'json');
}
// Version 详情展示切换
function toggleVersionInfo(el) {
	$(el).next().fadeToggle();
}
function closeVersionInfo(el) {
	$(el).parent().fadeToggle();
}
// Module Version 格式化
function moduleVersionFmt(value, row, index) {
	var tips = '',
		list = row.module_version_available || [],
		itemText = '';

	tips = [
		'<div style="color: blue;display: inline;padding: 0px 2px;cursor: pointer;" onclick=showModuleVersion('+JSON.stringify(list)+',event) >',
			'<span>['+list.length+']</span>',
		'</div>'
	].join('');

	if(list.length) {
		return value + tips;
	}else {
		return value;
	}
}

function showModuleVersion(list, evt) {
	var tips = '',
		list = list || [],
		itemText = '';

	itemText = list.map(function(item){
		
		return '<span class="item">'+item+'</span>';
	}).join('');
	
	tips += '<div class="version-details normal">' + itemText + '</div>';

	showPopLayer({event: evt, html: tips});
}

function softwareVersionFmt(value, row, index) {
	var tips = '',
		list = row.software_version_available || [],
		itemText = '';

	tips = [
		'<div class="cellNameClass" style="cursor: pointer;" onclick=showSoftwareVersion('+JSON.stringify(list)+',event) >',
			'<span class="el-icon el-icon-status-upgrading"></span>',
		'</div>'
	].join('');

	itemText = list.map(function(item){
		var sv = item.software_version,
			mv = item.module_version;
		
		if(mv && mv.length) {
			sv += ' [ ' + mv.join(', ') + ' ]'
		}
		
		return '<span class="item">'+sv+'</span>';
	}).join('');
	
	if(list.length) {
		return value + tips;
	}else {
		return value;
	}
}

function showSoftwareVersion(list,evt) {
	var tips = '',
		list = list || [],
		itemText = '';

	itemText = list.map(function(item){
		var sv = item.software_version,
			mv = item.module_version;
		
		if(mv && mv.length) {
			sv += ' [ ' + mv.join(', ') + ' ]'
		}
		
		return '<span class="item">'+sv+'</span>';
	}).join('');
	
	itemText = '<span class="item" style="font-weight: bold;"><%=rb.getString("SoftwareVersion")%> [<%=rb.getString("MoKuaiBanNen")%>]</span>' + itemText;

	tips += '<div class="version-details normal">' + itemText + '</div>';

	showPopLayer({event: evt, html: tips});
}
function modeFmt(value,row,index){
	if(value == 'fullband'){
		return "<%= rb.getString("BuSuoPin")%>";
	}else if(value == 'pcilock'){
		return "<%= rb.getString("SuoPCI")%>";
	}else if(value == 'freqpreferred'){
		return "<%= rb.getString("SuoPin")%>";
	}else if(value == 'pcionlylock'){
		return "PCI only lock";
	}
}
</script>