<%@ page contentType="text/html;charset=UTF-8"%>
	<%@ include file="/common/taglibs.jsp"%>
		<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
			<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
				<%
	UserInfo ui = (UserInfo) session.getAttribute(ComConstants.SESSION_KEY);
%>

					<style>
						/*SAS 操作页面*/
						.el-message--warning .el-icon-close{
							font-size:16px !important
						}
						
						.el-icon-close{
							font-size:30px !important
						}
						#procedureLogTableDiv {
							display: flex;
							height: 350px;
							width: 90%;
							margin-top:30px;
							padding-left: 50px;
						}
						
						.progressBox {
							display: flex;
							flex: 1;
							height: 290px;
							position: relative;
							margin-top: 30px;
							flex-direction: column;
							margin-bottom: 30px
						}
						
						.btnRadiusContainer {
							display: inline-block;
							height: 60px;
							width: 90px;
							vertical-align: middle;
							flex: 1;
						}
						
						.btnRadiusContainer span {
							display: inline-block;
							margin-top: 0px;
							width: 90px;
							text-align: center;
							font-size: 15px;
						}
						
						.btnRadiusContainer .btnRadius {
							width: 35px;
							height: 35px;
							border-radius: 35px;
							margin-left: 28px;
							margin-top: 10px;
						}
						
						.executed {
							background: url('${ctx}/css/images/sas/sas_successgreen.png') no-repeat center;
						}
						
						.nonexecution {
							background: url('${ctx}/css/images/sas/sas_successgary.png') no-repeat center;
						}
						
						.executing {
							background: url('${ctx}/css/images/sas/sas_successBlue.png') no-repeat center;
						}
						
						.topBtn {
							width: 35px;
							height: 35px;
							border-radius: 35px;
							margin-left: 47px;
							background: url('${ctx}/css/images/sas/sas_warning.png') no-repeat center !important;
							margin-top: 10px;
						}
						
						.topbtnRadiusContainer {
							height: 120px;
							width: 120px;
							vertical-align: middle;
							margin:auto;
							margin-bottom: 20px;
							display: none;
						}
						
						.bottomBtnGroup {
						
							display: flex;
							width: 990px;
							align-items: center;
							margin:auto
						}
						
						.topbtnRadiusContainer div span {
							display: inline-block;
							margin-top: 0px;
							width: 130px;
							text-align: center;
							font-size: 15px;
						}
						
						.line {
							flex: 2;
							display: inline-block;
							width: 140px;
							height: 2px;
							background: #B0CBDD;
							margin: 0px 10px;
							vertical-align: middle;
						}
						
						.actived_line {
							background: #92DBC7;
						}
						
						.prev_incline_line {
							position: relative;
							top: -36px;
							left: -161px;
							display: inline-block;
							width: 152px;
							height: 2px;
							background: #92DBC7;
							margin: 0px 10px;
							vertical-align: middle;
							transform: rotate(-30deg);
						}
						
						.next_incline_line {
							position: relative;
							top: -37px;
							left: 125px;
							display: inline-block;
							width: 152px;
							height: 2px;
							background: #92DBC7;
							margin: 0px 10px;
							vertical-align: middle;
							transform: rotate(30deg);
						}
						
						.verticalLine {
							display: inline-block;
							height: 50px;
							width: 2px;
							background: #92DBC7;
							margin: 5px 62px;
							vertical-align: middle;
						}
						
						.active_btn {
							background: url('${ctx}/css/images/sas/sas_reboot.png') no-repeat center !important;
							animation: rotateBgd 2s linear infinite;
							-moz-animation: rotateBgd 2s linear infinite;
							/* Firefox */
							-webkit-animation: rotateBgd 2s linear infinite;
							/* Safari 和 Chrome */
							-o-animation: rotateBgd 2s linear infinite;
							/* Opera */
						}
						
						@keyframes rotateBgd {
							from {
								transform: rotate(360deg);
								-moz-transform: rotate(360deg);
								-webkit-transform: rotate(360deg);
								-moz-transform: rotate(360deg);
							}
							to {
								transform: rotate(0deg);
								-moz-transform: rotate(0deg);
								-webkit-transform: rotate(0deg);
								-moz-transform: rotate(0deg);
							}
						}
						
						.active_line_prev {
							background: linear-gradient(to right, #83D5C0, #43B9DC, #20A5F9);
							background: -webkit-linear-gradient(left, #83D5C0, #43B9DC, #20A5F9);
						}
						
						.active_line_next {
							background: linear-gradient(to right, #20A5F9, #43B9DC, #83D5C0);
							background: -webkit-linear-gradient(left, #20A5F9, #43B9DC, #83D5C0);
						}
						
						.GrantedMenu {
							position: absolute;
							width: 160px;
							/*height: 70px;*/
							background: #FFFFFF;
							left: -30px;
							box-shadow: 1px 2px 10px #DEEEF9;
							-webkit-box-shadow: 1px 2px 10px #DEEEF9;
							display: none;
						}
						
						.GrantedMenu li {
							width: 143px;
							padding-left: 17px;
							border-bottom: 1px solid #FBFBFB;
							height: 34px;
							line-height: 34px;
							color: #A1A1A1;
							font-size: 12px;
						}
						
						.GrantedMenu li:hover {
							color: #1DA3FC;
							cursor: pointer;
						}
						
						.GrantedMenu li:last-child {
							border-bottom: none;
						}
						
						#procedureLogTableDiv .datagrid-view {
							height: 371px !important;
						}
						
						.searchSASType {
							display: block
						}
						.showSelect{
							margin-top:10px
						}
						.wline{
							background: #E9E9E9;
							width:100%;
							height:1px
						}
						.selectBox{
							width: 300px;
							margin: auto;
							height: 30px;
							line-height: 30px
						}
						.selectBox p{
							float:left;
							width:100px;
							border:#e9e9e9 solid 1px;
							text-align: center;
							cursor: pointer
						}
						.selectHover{
							border:solid 1px #1913BB !important;
							background:#F1F1FC;
							color: #1913BB
						}
						.selectHover p{
							
						}
					</style>
					<div id="procedure">
							<!-- 自动注册 手动部分 -->
					<div class="progressBox">
						<div class="group-title not-extend" style='margin-left:50px'> 
							<span class="title-icon"></span>
							<span class="title-text"><%=rb.getString("SASZhuangTai")%></span>
						</div>
						<div class="showSelect">
							<div class="selectBox">
								<p class="pcell" onclick="selectTab('pcell')">Pcell</p>
								<p class="scell" onclick="selectTab('scell')">Scell</p>
							</div>
						</div>
						<div class="topbtnRadiusContainer">
							<div style="position:relative">
								<div class="btnRadius topBtn" momentIndex="4"></div>
								<span class="proGrantedSpan" momentIndexSpan="4">Grant Suspended</span>
								<ul class="GrantedMenu">
									<li class="GrantedMenuItem" id="grantSuspendHeart">
										Heartbeat req
									</li>
									<li class="GrantedMenuItem" id="grantSuspenfRelin">
										Relinquishment req
									</li>
								</ul>
							</div>

							<span class="verticalLine"></span>
							<span class="prev_incline_line active_line_prev"></span>
							<span class="next_incline_line active_line_next"></span>

						</div>

						<div class="bottomBtnGroup">
							<div class="btnRadiusContainer" style="position: relative;">
								<div class="btnRadius active_btn" momentIndex="0"></div>
								<span class="proGrantedSpan" momentIndexSpan="0">Unregistered</span>
								<ul class="GrantedMenu">
									<li class="GrantedMenuItem" id="unResResgister">
										Register req
									</li>
								</ul>
							</div>
							<span class="line active_line_next"></span>

							<div class="btnRadiusContainer" style="position: relative;">
								<div class="btnRadius nonexecution" momentIndex="1"></div>
								<span class="proGrantedSpan" momentIndexSpan="1">Registered</span>
								<ul class="GrantedMenu">
									<li class="GrantedMenuItem" id="resDere">
										Deregister req
									</li>
									<li class="GrantedMenuItem" id="resGrant">
										Grant req
									</li>
								</ul>
							</div>
							<span class="line"></span>

							<div class="btnRadiusContainer" style="position: relative;">
								<div class="btnRadius nonexecution" momentIndex="3"></div>
								<span class="proGrantedSpan" momentIndexSpan="3" style="margin-bottom: 8px;">Granted</span>
								<ul class="GrantedMenu">
									<li class="GrantedMenuItem" id="grantHeart">
										Heartbeat req
									</li>
									<li class="GrantedMenuItem" id="grantRelin">
										Relinquishment req
									</li>
								</ul>
							</div>
							<span class="line"></span>

							<div class="btnRadiusContainer" style="position: relative;">
								<div class="btnRadius nonexecution" momentIndex="5">
									<!--<img src="img/loading.gif" alt="" />-->
								</div>
								<span class="proGrantedSpan" momentIndexSpan="5">Authorized</span>
								<ul class="GrantedMenu">
									<li class="GrantedMenuItem" id="AuthorizedHeart">
										Heartbeat req
									</li>
									<li class="GrantedMenuItem" id="AuthorizedRelin">
										Relinquishment req
									</li>
								</ul>
							</div>
							
							<!--<div class="btnRadiusContainer" style="position:relative;">
								<div class="btnRadius nonexecution" momentIndex="6"></div>
								<span class="proGrantedSpan" momentIndexSpan="6">Transmission</span>
								<ul class="GrantedMenu">
									<li class="GrantedMenuItem" id="transHeart">
										Heartbeat req
									</li>
									<li class="GrantedMenuItem" id="transRelin">
										Relinquishment req
									</li>
								</ul>
							</div>-->
						</div>
					</div>
					<div class="wline"></div>
					<!-- log 部分 -->
					<div id="procedureLogTableDiv" class="" style="height:470px;">
						<div class="tableBar" id="sasProceDureLog_toolabr">
							<div class="group-title not-extend">
								<span class="title-icon"></span>
								<span class="title-text"><%=rb.getString("RiZhi")%></span>
							</div>
							<span class="el-icon el-icon-operation-export" title='<%=rb.getString("DaoChu")%>' style="float:right;margin-top:-18px;margin-right:25px" onclick="sasExportProcedureLogs();"></span>
							<span class="el-icon el-icon-operation-delete CODE_ADVANCE_SAS hidden" title='<%=rb.getString("ShanChu")%>'style="float:right;margin-top:-18px;" onclick="sasClearProcedureLogs();"></span>
						</div>
						<table id="procedureTable" class="panelTableDiv"></table>
					</div>
					<form method="post" style="display: none" id="sasProcedureLogs"></form>
					</div>
				
					<script>
							var hasChooseSn;
							var dualCarrierType = 'pcell';
							var deviceType;
							var tabName;
							var tabNames;
							new Vue();
								new Vue({
									el: '#procedure',
									data(){
										
									},
									computed: {
										
									},
									methods: {
										init(data,type){
											
										}
									},
									mounted(){
										eventBus.$off('open-dialog').$on('open-dialog', this.init);
									}
								});

							$(function () {
								//sas 状态操作  每6秒刷新
								var seriNum = "${serialNumber}";
								hasChooseSn = "${serialNumber}";
								deviceType =$('#deviceTypeSelect').val() || 'eNB'
								tabName = $('#tabNames').val() || '1' // 1是真实CBSD 2 是虚拟
								if(tabName !=='1'){
									$('.GrantedMenu').css('display','none')
								}
								/* prodecure log table 加载 */
								var prodecureLogData = {
									"total": 1,
									"rows": [{
										serial_number: '20916781896791',
										cbsdId: '234dr54345',
										status: 'isdonging',
										timeZone: '1992.11.28',
										message: 'this is one message'
									}],
									"footer": [],
									"properties": null

								}
								if (deviceType == 'eNB') {
									sasType(seriNum);
								}else{
									dualCarrierType= '';
									$('.showSelect').css('display','none')
								}
								$("#procedureTable").datagrid({
									border: false,
									fit: true,
									url: '${ctx}/cell/SAS/getSasCbsdLog.action',
									rownumber: false,
									striped: true,
									queryParams: {
										serialNumber: seriNum,
										dualCarrierType: dualCarrierType
									},
									toolbar: '#sasProceDureLog_toolabr',
									singleSelect: true,
									fitColumns: true,
									pagination: true,
									pagePosition: 'bottom',
									idField: 'id',
									columns: [[
										{ field: 'serialNumber', width: 50, title: '<%=rb.getString("XiaoZhanBianMa")%>' },
										{ field: 'cbsdId', width: 50, title: '<%=rb.getString("CBSDID")%>' },
										{ field: 'state', width: 20, formatter: zhuangtaiFormatter, title: '<%=rb.getString("SASZhuangTai")%>' },
										{ field: 'time', width: 50, title: '<%=rb.getString("ShiJian")%>' },
										{ field: 'message', width: 150, formatter: titleFormatter, title: '<%=rb.getString("XinXi")%>' },
									]],
									onLoadSuccess: loadSuccess_SASprocedureTable
								})

								setIntervalFun();
								clearInterval(SASProcressInterval);
								clearInterval(datagridIntelVal);
								SASProcressInterval = setInterval(setIntervalFun, 6000)
								datagridIntelVal = setInterval(function () {
									$("#procedureTable").datagrid("reload");
								}, 6000)

							})
							function sasType(num) {
								var params = {
									serialNumber: num,
								}
								if (tabName != "1"){
									params.isVirtual = "1";
								}
	
								$.post("${ctx}/cell/SAS/isSupportMultiInstallParams.action", params, function (data) {
									if (data["supportMultiInstallParams"]) {
										dualCarrierType = 'pcell'
										$('.showSelect').css('display','block')
										$('.pcell').addClass('selectHover')

									} else {
										$('.showSelect').css('display','none')
										dualCarrierType = ''
									}
								}, "json");
							}
							function selectTab(data){
								if(data === 'pcell'){
 									$('.pcell').addClass('selectHover')
									$('.scell').removeClass('selectHover')
									dualCarrierType = 'pcell'
									var opt = $("#procedureTable").datagrid("options"),
										params = opt.queryParams;
									$("#procedureTable").datagrid("reload", Object.assign(params, { dualCarrierType: dualCarrierType }));
									setIntervalFun()
								}else{
									dualCarrierType = 'scell'
									var opt = $("#procedureTable").datagrid("options"),
										params = opt.queryParams;
									$("#procedureTable").datagrid("reload", Object.assign(params, { dualCarrierType: dualCarrierType }));
									$('.scell').addClass('selectHover')
									$('.pcell').removeClass('selectHover')
									setIntervalFun()
								}
								
							}
							function setIntervalFun() {
								var procedureCtn = $("#procedure");
								
								if(!procedureCtn.length) {
									clearInterval(SASProcressInterval);
									clearInterval(datagridIntelVal);
									
									return;
								}
								
								var newdeviceType = deviceType.toLowerCase()
								var url=''
								if(tabName== '1'){
									url = "${ctx}/cell/SAS/getState.action?serialNumber=" + hasChooseSn + '&dualCarrierType=' + dualCarrierType +'&deviceType=' + newdeviceType
								}else{
									url = "${ctx}/cell/SAS/getState.action?serialNumber=" + hasChooseSn + '&dualCarrierType=' + dualCarrierType +'&deviceType=' + newdeviceType +'&virtualCbsd=1'
								}
								$.post(url, function (data) {
									var momentStatus = data["state"];
									//根据返回的momentStatus  添加响应的操作
									var objBtn = $("[momentIndex=" + momentStatus + "]");
									if (data["isDoing"]) { //如果正在执行  
										$(".proGrantedSpan").off('mouseenter');
										$(".GrantedMenu").hide();
									} else if(tabName =='1') {
										//先解绑当前鼠标划入事件
										$(".proGrantedSpan").off('mouseenter');
										$(".GrantedMenu").hide();
										//绑定划入出现菜单事件
										$(".proGrantedSpan[momentIndexSpan=" + momentStatus + "]").on("mouseenter", function (e) {
											$(this).next(".GrantedMenu").fadeIn(300);

										});
										//如果自动执行有问题   需要手动添加  
										//给菜单绑定点击事件s
										$(".GrantedMenuItem").off("click");
										$(".GrantedMenuItem").on("click", function (e) {
											var status = $('#deviceTypeSelect').val() || 'eNB'
											var params = {};
											params.dualCarrierType = dualCarrierType;
											params.serialNumber = hasChooseSn;
											params.deviceType = status.toLowerCase();
											var targetId = e.target.id;
											switch (targetId) {
												case 'grantHeart':
													params.action = "3";
													break;
												case 'grantRelin':
													params.action = "4";
													break;
												case 'unResResgister':
													params.action = "0";
													break;
												case 'resDere':
													params.action = "1";
													break;
												case 'resGrant':
													params.action = "2";
													break;
												case 'grantSuspendHeart':
													params.action = "3";
													break;
												case 'grantSuspenfRelin':
													params.action = "4";
													break;
												case 'AuthorizedHeart':
													params.action = "3";
													break;
												case 'AuthorizedRelin':
													params.action = "4";
													break;
												case 'transRelin':
													params.action = "4";
													break;
												case 'transHeart':
													params.action = "3";
													break;
											}

											$.post("${ctx}/cell/SAS/operate.action", params, function (data) {
												if (data["success"]) {

												} else {
													showMsg('error_msg', data["message"]);
													return false;
												}
											}, "json")
											var that = $(this).parent(".GrantedMenu").siblings(".btnRadius");
											that.addClass("active_btn");
											that.parent(".btnRadiusContainer").siblings().find(".btnRadius").removeClass("active_btn").addClass("executed");
											that.parents().siblings(".line").removeClass("active_line_next active_line_prev");
											that.parent().next().addClass("active_line_next");
											that.parent().prev().addClass("active_line_prev");
											$(this).parent(".GrantedMenu").fadeOut(300);
										})
									}
									//当前的状态为  

									if (momentStatus == 4) {
										$(".topbtnRadiusContainer").css("display", "block");
										$(".btnRadiusContainer .btnRadius[momentIndex!=0],.btnRadiusContainer .btnRadius[momentIndex!=1]").removeClass("nonexecution executing executed active_btn").addClass("nonexecution");
										$(".btnRadiusContainer .btnRadius[momentIndex=0]").removeClass("nonexecution executing executed active_btn").addClass("executed");
										$(".btnRadiusContainer .btnRadius[momentIndex=0]").parent().next().removeClass("active_line_next active_line_prev").addClass("actived_line");
										$(".btnRadiusContainer .btnRadius[momentIndex=0]").parent().next().siblings(".line").removeClass("active_line_next active_line_prev actived_line");
										$(".btnRadiusContainer .btnRadius[momentIndex=1]").removeClass("nonexecution executing executed").addClass("executed");

									} else {
										$(".topbtnRadiusContainer").css("display", "none");
										$(".btnRadiusContainer .btnRadius").removeClass("nonexecution executing executed active_btn").addClass("nonexecution")
										$(".btnRadiusContainer .btnRadius:lt(" + momentStatus + ")").removeClass("nonexecution executing executed active_btn").addClass("executed");
										if (momentStatus == 5) {
											$(".btnRadiusContainer .btnRadius:last").removeClass("nonexecution executing executed active_btn").addClass("nonexecution");
										}
										if (data["isDoing"]) {
											objBtn.removeClass("nonexecution executing executed active_btn").addClass("active_btn");
										} else {
											objBtn.removeClass("nonexecution executing executed active_btn").addClass("executing");
										}
										/*objBtn.removeClass("nonexecution executing executed active_btn").addClass("executing");*/
										objBtn.parents().prevAll(".line").removeClass("active_line_next active_line_prev").addClass("actived_line");
										objBtn.parents().nextAll(".line").removeClass("active_line_next active_line_prev actived_line");
										objBtn.parent().next().addClass("active_line_next");
										objBtn.parent().prev().addClass("active_line_prev");
									}
									//return setIntervalFun;
								}, "json");
							}

							//关闭procedure页面并关闭页面的定时器
							function closeProcedure() {
								$("#procedurePanel").slideUp(600, function () {
									clearInterval(SASProcressInterval);
									clearInterval(datagridIntelVal);
									//$("#procedurePanel").panel("destroy");
								});

							}
							//导出单个基站注册日志   
							function sasExportProcedureLogs() {
								var params = {
									serialNumber: hasChooseSn,
									timeZone: timeZone
								}
								/* $("#sasProcedureLogs").form('submit', {
									url: "${ctx}/cell/SAS/exportSasCbsdLog.action",
									onSubmit: function (param) {
										param.serialNumber = hasChooseSn,
											param.timeZone = timeZone,
											param.carrierType = carrierType
									}
								}); */
								exportByForm("${ctx}/cell/SAS/exportSasCbsdLog.action",{
									serialNumber: hasChooseSn,
									dualCarrierType: dualCarrierType
								});
							}
							//清空单个基站注册日志
							function sasClearProcedureLogs() {
								var params = {
									serialNumber: hasChooseSn,
									timeZone: timeZone,
									dualCarrierType: dualCarrierType
								}
								$.messager.confirm('<%=rb.getString("QueRen")%>', '<%=rb.getString("QueRenQingKongSuoYouRiZhi")%>', function (r) {
									if (r) {
										$.post("${ctx}/cell/SAS/clearSasCbsdLog.action", params, function (data) {
											if (data["success"]) {
												$("#procedureTable").datagrid("reload")
											} else {
												showMsg('error_msg', data["message"]);
											}
										}, "json")
									}
								}).addClass("seriousConfirm");

							}
							function loadSuccess_SASprocedureTable() {
								$(this).datagrid("fixRownumber");
								$(this).datagrid("enableContextmenuAutoSize");
							}

							function titleFormatter(val, row, index) {
								value = '<div style="text-overflow:ellipsis;overflow:hidden;" title=" ' + val + ' "> ' + val + '</div>'
								return value;
							}
							function zhuangtaiFormatter(value,rowData,index){
								if(value == "0"){
									value = "Unregistered";
								}else if(value == "1"){
									value = "Registered";
								}else if(value == "3"){
									value = "Granted";
								}else if(value == "5"){
									value = "Authorized"
								}else if(value == "4"){
									value = "Grant Suspended"
								}else if(value == "6"){
									value = "Transmission"
								}else{
									value = ""
								}

								if(rowData.deviceType === 'CPE' && rowData.cpeSasStatusEqualOmc == '0'){
									value = "<span class='el-icon el-icon-circle-warning' style='font-size:22px'>"+ "<span style='font-size:12px'>"+(value)+"</span>"+"</span>"
								}

								return value;
							}

						</script>