<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<style type="text/css">
	#deviceMigrationPage .el-table__body{
		width: 100% !important;
	}
	.defaultCls{
		position: relative;
		background-color: #F6F7FB;
		height: 100%;
	}
	.deviceMigrationMainBox{
		position: absolute;
		top: 20px;
		right: 30px;
		bottom: 20px;
		left: 30px;
		height: 100%;
	}
	.migrationMainTable{
		position: relative;
		top:20px;
		height: 90%;
	}
	.migrationConfirmBox{
		position: relative;
		top:20px;
		height: 90%;
	}
	.migrationResultBox{
		position: relative;
		top:20px;
		height: 90%;
	}
	.titleCls{
		font-size: 18px;
		font-weight: 550;
		color: #333333;
		margin-bottom: 20px;
	}
	.migrationTableBox{
		display: flex;
		height: 90%;
	}
	.tabListMain{
		height: calc( 100% - 95px);
	}
	.commercialTableList{
		height: 100%;
		width: 49.5%;
		background-color: #FFFFFF;
		border:1px solid #E9E9E9;
	}
	.tableSpacing{
		height: 100%;
		width: 1%;
	}
	.betaTableList{
		height: 100%;
		width: 49.5%;
		background-color: #FFFFFF;
		border:1px solid #E9E9E9;
	}
	.tableHeaderCls{
		height: 40px;
		border-bottom:1px solid #E9E9E9;
		width: 100%;
		text-align: center;
		line-height: 40px;
		font-size: 16px;
		font-weight: 550;
		position: relative;
		box-sizing: border-box;
	}
	.selectBoxCls{
		position: absolute;
		right: 10px;
		top: 0px;
	}
	.selectMain{
		position: relative;
		height: 40px;
		display: flex;
		align-items: center;
		font-size: 14px;
		font-weight: 500;
	}
	.selectMain .el-icon::before{
		font-size: 16px;
	}
	.commercialSelectTable{
		position: absolute;
		top: 38px;
		right: -30px;
		width: 628px;
		height: 400px;
		background-color: #FFFFFF;
		border: 1px solid #DCDFE6;
		box-shadow: 0px 0px 15px rgba(0, 0, 0, 0.15);
		border-radius: 4px;
		z-index: 99
	}
	.betaSelectTable{
		position: absolute;
		top: 38px;
		right: -30px;
		width: 628px;
		height: 400px;
		background-color: #FFFFFF;
		border: 1px solid #DCDFE6;
		box-shadow: 0px 0px 15px rgba(0, 0, 0, 0.15);
		border-radius: 4px;
		z-index: 99
	}
	.selectBoxTitle{
		height: 45px;
		width: 100%;
		padding-left: 20px;
		border-bottom: 1px solid #E9E9E9;
		text-align:start;
		line-height: 45px;
		font-size: 16px;
		font-weight: bold;
		color: #333333;
		box-sizing: border-box;
		position: relative;
	}
	.selectBoxTitle .el-icon::before{
		font-size: 16px;
	}
	.tabTitleCls{
		padding: 0 10px;
	}
	.selectBoxMain{
		margin: 0px 20px;
	}
	
	.tableInfoCls .el-ctable{
		border-bottom: none;
	}
	.tableInfoCls .tableInfoHeader{
		height: 40px;
		line-height: 40px;
		position: relative;
		padding-left: 10px;
		border-top: 1px  solid #F4F4F4;
	}
	.tableInfoCls .el-ctable-toolbar{
		padding: 0px !important;
	}
	.tableInfoCls .el-table__header-wrapper{
		display: none;
	}
	.tableInfoHeader div:nth-child(1){
		text-align: start;
		font-size: 14px;
		font-weight: bold;
		color: #2E2E2E;
	}
	.tableInfoHeader div:nth-child(2){
		position: absolute;
		right: 0px;
		top: 0px;
		font-size: 12px;
		font-weight: bold;
		color: #000000;
		cursor: pointer;
	}
	.tableItemCls{
		width: 100%;
	}
	.tableItemCls .el-icon::before{
		font-size: 18px !important;
	}
	.tableItemCls .item_show{
		display: none;
		float: right;
		padding-top: 5px;
	}
	.tableItemCls:hover .item_show{
		display: inline-block;
	}
	.tableFooter{
		height: 48px;
		line-height: 40px;
		display: flex;
		align-items: center;
		padding-left: 30px;
		border-top:1px solid #E9E9E9;
	}
	.tableFooter div{
		padding-top:14px;
	}
	#deviceMigrationPage .el-tabs--top{
		height: 100%;
	}
	#deviceMigrationPage .queryContainer{
		height: 28px;
		line-height: 28px;
		margin-left: 0px;
	}
	#deviceMigrationPage .el-input--small .el-input__inner{
		height: 28px;
		line-height: 28px;
	}
	#deviceMigrationPage .productTypeBox .el-input--mini .el-input__inner{
		height: 30px;
		line-height: 30px;
	}
	#deviceMigrationPage .productTypeBox .el-input--mini{
		width: 120px;
		margin-left: 15px;
	}
	.confirmTableBox{
		height: calc( 100% - 95px);
		width: 100%;
		background-color: #FFFFFF;
		border:1px solid #E9E9E9;
		overflow: auto;
	}
	.confirmTableBoxMain{
		margin: 20px 40px;
		border: 1px solid #E9E9E9;
		height: calc( 100% - 160px);
	}
	.resultTableBox{
		height: 90%;
		width: 100%;
		background-color: #FFFFFF;
		border:1px solid #E9E9E9;
	}
	.resultTableBoxMain{
		position: relative;
		height: 100%;
		overflow: hidden;
	}
	.confirmTableInfoCls{
		height: 100%;
	}
	.confirmTableInfoCls .icon_show{
		display: none;
		float: right;
		padding-top: 5px;
	}
	.confirmTableInfoCls .el-table__row:hover  .icon_show{
		display: inline-block;
	}
	.confirmBoxClearBtn{
		position: absolute;
		right: 15px;
		top: 10px;
		font-weight: bold;
		font-size: 12px;
		cursor: pointer;
	}
	.migrationMainTable .advanceQuery,resultTableBoxInfo .advanceQuery{
		margin-left: 10px;
		height: 28px;
	}
	.confirmPrompt{
		margin: 20px 40px 0px 40px;
		color: #333333;
		font-size: 14px;
	}
	#deviceMigrationPage  .resultHeadBox{
		position: absolute;
		top:12px;
		right: 50px;
		display: flex;
		z-index: 99;
	}
	#deviceMigrationPage  .resultHeadBox .statisticsSuccessDiv{
		border: 1px solid #67D972;
		height: 30px;
		border-radius: 4px;
		display: flex;
		margin-right: 15px;
		align-items: center;
		box-sizing: border-box;
		overflow: hidden;
	}
	#deviceMigrationPage  .resultHeadBox .statisticsFailDiv{
		border: 1px solid #E88282;
		height: 30px;
		border-radius: 4px;
		display: flex;
		align-items: center;
		margin-right: 15px;
		box-sizing: border-box;
		overflow: hidden;
	}
	.statisticsSuccessDiv .el-icon::before{
		font-size: 16px;
		color:#67D972;
	}
	.statisticsSuccessDiv > div:first-child{
		padding: 0px 10px;
		color: #67D972;
		height: 30px;
		display: flex;
		align-items: center;
		border-radius: 4px;
		font-size: 12px;
		background-color: #EEFFF3;
	}
	.statisticsSuccessDiv > div:last-child{
		padding: 0px 20px;
		height: 30px;
		line-height: 30px;
		border-left: 1px solid #67D972;
	}
	.statisticsFailDiv .el-icon::before{
		font-size: 16px;
		color: #E88282;
	}
	.statisticsFailDiv > div:first-child{
		padding: 0px 10px;
		color: #E88282;
		height: 30px;
		display: flex;
		align-items: center;
		border-radius: 4px;
		font-size: 12px;
		box-sizing: border-box;
		background-color: #FEF2F2;
		
	}
	.statisticsFailDiv > div:last-child{
		padding: 0px 20px;
		height: 30px;
		line-height: 30px;
		border-left: 1px solid #E88282;
	}
	.resultEnbAndCpeTabs{
		position: absolute;
		top: 50px;
		left: 30px;
		display: flex;
		z-index: 30;
	}
	.resultEnbAndCpeTabs div{
		box-sizing: border-box;
		cursor: pointer;
	}
	.tabEnbDefaultCls{
		height: 30px;
		padding: 0px 20px;
		display: flex;
		align-items: center;
		border: 1px solid #E9E9E9;
		border-right: none;
		border-radius: 4px 0px 0px 4px;
		color: #666666;
	}
	.tabEnbDefaultCls .el-icon::before,.tabcpeDefaultCls .el-icon::before{
		color: #666666;
	}
	.tabEnbSelectCls{
		height: 30px;
		padding: 0px 20px;
		display: flex;
		align-items: center;
		background-color: #4D84FF;
		border: 1px solid #4D84FF;
		border-left: none;
		border-radius: 4px 0px 0px 4px;
		color: #FFFFFF;
	}
	.tabEnbSelectCls .el-icon::before,.tabcpeSelectCls .el-icon::before{
		color: #FFFFFF;
	}
	.tabcpeDefaultCls{
		height: 30px;
		padding: 0px 20px;
		display: flex;
		align-items: center;
		border: 1px solid #E9E9E9;
		border-left: none;
		border-radius: 0px 4px 4px 0px;
		color: #666666;
	}
	.tabcpeSelectCls{
		height: 30px;
		padding: 0px 20px;
		display: flex;
		align-items: center;
		border: 1px solid #4D84FF;
		border-left: none;
		border-radius: 0px 4px 4px 0px;
		background-color: #4D84FF;
		color: #FFFFFF;
	}
	.tabHeaderBtn{
		position: absolute;
		top: 52px;
		right: 20px;
		z-index: 99;
	}
	.enbListTable{
		position: absolute;
		top: 40px;
		bottom: 0px;
		left: 0px;
		right: 0px;
		z-index: 20;
	}
	.cpeListTable{
		position: absolute;
		top: 40px;
		bottom: 0px;
		left: 0px;
		right: 0px;
		z-index: 20;
	}
	.activeStatusItem .el-icon,.inactiveStatusItem .el-icon, .statusItemInProgress .el-icon, .statusItemSuccess .el-icon, .statusItemFail .el-icon{
		font-size:20px;
		vertical-align:bottom;
		margin-right:5px;
	}
	.activeStatusItem .el-icon::before, .statusItemSuccess .el-icon::before{
		color:#67D972;
	}
	.inactiveStatusItem .el-icon::before,.statusItemFail .el-icon::before{
		color:#E88282;
	}
</style>
<div class="defaultCls" id="deviceMigrationPage" >
	<div class="deviceMigrationMainBox">
		<div class="migrationMainTable" v-show="deviceMigrationMainShow">
			<div class="titleCls"><%=rb.getString("SheBeiQianYi")%></div>
			<div class="circleButton placeholder-bt" placeholder="<%=rb.getString("JieGuo")%>"  style="position: absolute;top:0px;right:0px;">		
				<span class="el-icon el-icon-circle-result" @click="goResult"></span>
			</div>
			<div class="migrationTableBox">
				<div class="commercialTableList">
					<div class="tableHeaderCls">
						Commercial Cloud
						<div class="selectBoxCls">
							<div class="selectMain">
								<span>Selected</span>
								<span style="color:#4D84FF;margin:0px 5px;">({{commercialSelectNum}})</span>
								<span v-show="!commercialSelectShow" class="el-icon el-icon-circle-down" @click="openCommercialSelectTable"></span>
								<span class="el-icon el-icon-circle-up" v-show="commercialSelectShow" @click="closeCommercialSelectTable"></span>
								<div class="commercialSelectTable" v-show="commercialSelectShow">
									<div class="selectBoxTitle">
										<span>Selected Device</span>
										<span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeCommercialSelectTable"></span>
									</div>
									<div class="selectBoxMain">
										 <el-tabs class="fit" v-model="commercialSelectActiveName">
            								<el-tab-pane name="enb">
												<span slot="label" class="tabTitleCls">eNB</span>
												<div class="tableInfoCls">
													<div class="tableInfoHeader">
														<div>Serial Number+Cell Name</div>
														<div @click="clearCommercialEnbSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
													</div>
													<el-ctable 
														id="commercialEnbSelectTable" 
														ref="commercialEnbSelectTable" 
														:data="commercialEnbSelectList" 
														:showHeader="false"
														:rownumber="false"
														:front-pagination="true"
														height="270px" pagination="true" >
														<el-table-column prop="id" v-if="false"></el-table-column>
														<el-table-column width="588">
															<template slot-scope="scope" >
																<div class="tableItemCls">
																	<span>{{scope.row.snAndCellName}}</span>
																	<span @click="delCommercialEnbSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
																</div>
															</template>
														</el-table-column>
													</el-ctable>
												</div>
											</el-tab-pane>
											<el-tab-pane name="cpe">
												<span slot="label" class="tabTitleCls">CPE</span>
												<div class="tableInfoCls">
													<div class="tableInfoHeader">
														<div>MAC Address+CPE Name</div>
														<div @click="clearCommercialCpeSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
													</div>
													<el-ctable 
														id="commercialCpeSelectTable" 
														ref="commercialCpeSelectTable" 
														:data="commercialCpeSelectList" 
														:showHeader="false"
														:rownumber="false"
														:front-pagination="true"
														:height="height" pagination="true" >
														<el-table-column prop="id" v-if="false"></el-table-column>
														<el-table-column width="588">
															<template slot-scope="scope" >
																<div class="tableItemCls">
																	<span>{{scope.row.macAndCPEName}}</span>
																	<span @click="delCommercialCpeSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
																</div>
															</template>
														</el-table-column>
													</el-ctable>
												</div>
											</el-tab-pane>
										 </el-tabs>
									</div>
								</div>
							</div>
							
						</div>
					</div>
					<div class="tabListMain">
						<el-tabs class="fit" v-model="commercialActiveName">
							<el-tab-pane name="enb">
								<span slot="label" class="tabTitleCls">eNB</span>
								<!--:url="commercialEnbTableUrl"  :data="enbTableList" -->
								 <el-ctable
								 	id="commercialEnbTable"
									:url="commercialEnbTableUrl"
									:query-params="commercialEnbQueryParams" 
									ref="commercialEnbTable"
									@selection-change='commercialEnbSelectChange'
									height="100%"
									time=6
									row-key="serial_number"
									:page-size="pageSize" 
									:page-list="pageList"
									:rownumber="true"
									pagination="true">
									<!-- 列表toolbar -->
									<template slot="toolbar">
										<div style="display:flex;">
											<div class="productTypeBox">
												<el-select v-model="commercialEnbQueryParams.product" size="mini" placeholder="<%=rb.getString("ChanPinLeiXing")%>" filterable>
													<el-option v-for="item in commercialEnbProductList" :label="item.text" :value="item.id">
													</el-option>
												</el-select>
											</div>
											<el-query type="normal" @query="queryCommercialEnbTable" placeholder="<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%>"></el-query>
										</div>
									</template>
										<!-- 列表columns -->
									<el-table-column type='selection' width="45" :selectable="checkSelectTable" :reserve-selection="true"></el-table-column>
									<el-table-column prop="connection_status" width="50">
										<template slot-scope="scope">
											<div :class="{
												'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
												'':scope.row.have_connected==2,
												'conn_exc':scope.row.connection_status=='Exception',
												'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
										</template>
									</el-table-column>
									<el-table-column prop='serial_number' min-width="140" label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
									<el-table-column prop='host_name' min-width="120" label='<%=rb.getString("HostName")%>'></el-table-column>
									<el-table-column prop='op_state' min-width="120" label='<%=rb.getString("ShiFouJiHuo")%>'>
										<template slot-scope="scope">
											<div class='activeStatusItem' v-show="scope.row.op_state == '1'">
												<span class='el-icon el-icon-status-active' style='margin-right: 10px;'></span><%=rb.getString("JiHuo")%>
											</div>
											<div class='inactiveStatusItem' v-show="scope.row.op_state == '0'">
												<span class='el-icon el-icon-status-active' style='margin-right: 10px;'></span><%= rb.getString("QuJiHuo")%>
											</div>
										</template>
									</el-table-column>
									<el-table-column prop='product' min-width="120" label='<%=rb.getString("ChanPinLeiXing")%>'></el-table-column>
									<el-table-column prop='group_name' min-width="120" label='<%=rb.getString("SheBeiZu")%>'></el-table-column>
								</el-ctable>
								
							</el-tab-pane>
							<el-tab-pane name="cpe">
								<span slot="label" class="tabTitleCls">CPE</span>
								<el-ctable
								 	id="commercialCpeTable"
									:url="commercialCpeTableUrl"
									:query-params="commercialCpeQueryParams" 
									ref="commercialCpeTable" 
									height="100%"
									time=6
									row-key="macaddress"
									@selection-change='commercialCpeSelectChange'
									:page-size="pageSize" 
									:page-list="pageList" 
									pagination="true">
										<!-- 列表toolbar -->
									<template slot="toolbar">
										<div style="display:flex;">
											<div class="productTypeBox">
												<el-select v-model="commercialCpeQueryParams.product" size="mini" placeholder="<%=rb.getString("ChanPinXingHao")%>" filterable>
													<el-option v-for="item in commercialCpeProductList" :label="item.text" :value="item.id">
													</el-option>
												</el-select>
											</div>
											<el-query type="normal" @query="queryCommercialCpeTable" placeholder="<%=rb.getString("CPEBianMa")%> / <%=rb.getString("CPEName")%> / IMSI / MAC"></el-query>
										</div>
										
									</template>
										<!-- 列表columns -->
									<el-table-column type='selection' width="45" :selectable="checkSelectTable" :reserve-selection="true"></el-table-column>
									<el-table-column prop="connection_status" width="45">
										<template slot-scope="scope">
											<div :class="{
												'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
												'':scope.row.have_connected==2,
												'conn_exc':scope.row.connection_status=='Exception',
												'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
										</template>
									</el-table-column>
									<el-table-column prop='serial_number' min-width="140" label='<%=rb.getString("CPEBianMa")%>'></el-table-column>
									<el-table-column prop='cpe_name' min-width="100" label='<%=rb.getString("CPEName")%>'></el-table-column>
									<el-table-column prop='imsi' min-width="60" label='<%=rb.getString("IMSI")%>'></el-table-column>
									<el-table-column prop='macaddress' min-width="80" label='MAC'></el-table-column>
									<el-table-column prop='model_name' min-width="120" label='<%=rb.getString("ChanPinXingHao")%>' width='122'></el-table-column>
									<el-table-column prop='group_name' min-width="120" label='<%=rb.getString("SheBeiZu")%>'></el-table-column>
									<el-table-column prop='host_name' min-width="120" label='<%=rb.getString("HostName")%>'></el-table-column>
								</el-ctable>
							</el-tab-pane>
						</el-tabs>
					</div>
					<div class="tableFooter">
						<div>
							<el-button type="primary" size="mini" @click="commercialMigrationClick" :disabled="commercialMigrationDisabled"><%=rb.getString("YiDongDaoBeta")%></el-button>
							<el-button size="mini" @click="commercialMigrationCancel"><%=rb.getString("QuXiao")%></el-button>
						</div>
					</div>
				</div>
				<div class="tableSpacing"></div>
				<div class="betaTableList">
					<div class="tableHeaderCls">
						Beta Cloud
						<div class="selectBoxCls">
							<div class="selectMain">
								<span>Selected</span>
								<span style="color:#4D84FF;margin:0px 5px;">({{betaSelectNum}})</span>
								<span v-show="!betaSelectShow" class="el-icon el-icon-circle-down" @click="openBetaSelectTable"></span>
								<span class="el-icon el-icon-circle-up" v-show="betaSelectShow" @click="closeBetaSelectTable"></span>
								<div class="betaSelectTable" v-show="betaSelectShow">
									<div class="selectBoxTitle">
										<span>Selected Device</span>
										<span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBetaSelectTable"></span>
									</div>
									<div class="selectBoxMain">
										 <el-tabs class="fit" v-model="betaSelectActiveName">
            								<el-tab-pane name="enb">
												<span slot="label" class="tabTitleCls">eNB</span>
												<div class="tableInfoCls">
													<div class="tableInfoHeader">
														<div><%=rb.getString("XiaoZhanBianMa")%>+<%=rb.getString("HostName")%></div>
														<div @click="clearBetaEnbSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
													</div>
													<el-ctable 
														id="betaEnbSelectTable" 
														ref="betaEnbSelectTable" 
														:data="betaEnbSelectList" 
														:showHeader="false"
														:rownumber="false"
														:front-pagination="true"
														height="270px" pagination="true" >
														<el-table-column prop="id" v-if="false"></el-table-column>
														<el-table-column width="588">
															<template slot-scope="scope" >
																<div class="tableItemCls">
																	<span>{{scope.row.snAndCellName}}</span>
																	<span @click="delBetaEnbSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
																</div>
															</template>
														</el-table-column>
													</el-ctable>
												</div>
											</el-tab-pane>
											<el-tab-pane name="cpe">
												<span slot="label" class="tabTitleCls">CPE</span>
												<div class="tableInfoCls">
													<div class="tableInfoHeader">
														<div>MAC+<%=rb.getString("CPEName")%></div>
														<div @click="clearBetaCpeSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
													</div>
													<el-ctable 
														id="betaCpeSelectTable" 
														ref="betaCpeSelectTable" 
														:data="betaCpeSelectList" 
														:showHeader="false"
														:rownumber="false"
														:front-pagination="true"
														:height="height" pagination="true" >
														<el-table-column prop="id" v-if="false"></el-table-column>
														<el-table-column width="588">
															<template slot-scope="scope" >
																<div class="tableItemCls">
																	<span>{{scope.row.macAndCPEName}}</span>
																	<span @click="delBetaCpeSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
																</div>
															</template>
														</el-table-column>
													</el-ctable>
												</div>
											</el-tab-pane>
										 </el-tabs>
									</div>
								</div>
							</div>
							
						</div>
					</div>
					<div class="tabListMain">
						<el-tabs class="fit" v-model="betaActiveName">
							<el-tab-pane name="enb">
								<span slot="label" class="tabTitleCls">eNB</span>
								<!--:url="betaEnbTableUrl"  :data="enbTableList" -->
								 <el-ctable
								 	id="betaEnbTable"
									:url="betaEnbTableUrl"
									:query-params="betaEnbQueryParams" 
									ref="betaEnbTable"
									@selection-change='betaEnbSelectChange'
									height="100%" 
									time=6
									row-key="serial_number"
									:page-size="pageSize" 
									:page-list="pageList"
									:rownumber="true"
									pagination="true">
										<!-- 列表toolbar -->
									<template slot="toolbar">
										<div style="display:flex;">
											<div class="productTypeBox">
												<el-select v-model="betaEnbQueryParams.product" size="mini" placeholder="<%=rb.getString("ChanPinLeiXing")%>" filterable>
													<el-option v-for="item in betaEnbProductList" :label="item.text" :value="item.id">
													</el-option>
												</el-select>
											</div>
											<el-query type="normal" @query="queryBetaEnbTable" placeholder="<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%>"></el-query>
										</div>
									</template>
										<!-- 列表columns -->
									<el-table-column type='selection' width="45" :selectable="checkSelectTable" :reserve-selection="true"></el-table-column>
									<el-table-column prop="connection_status" width="50">
										<template slot-scope="scope">
											<div :class="{
												'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
												'':scope.row.have_connected==2,
												'conn_exc':scope.row.connection_status=='Exception',
												'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
										</template>
									</el-table-column>
									<el-table-column prop='serial_number' min-width="140" label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
									<el-table-column prop='host_name' min-width="120" label='<%=rb.getString("HostName")%>'></el-table-column>
									<el-table-column prop='op_state' min-width="120" label='<%=rb.getString("ShiFouJiHuo")%>'>
										<template slot-scope="scope">
											<div class='activeStatusItem' v-show="scope.row.op_state == '1'">
												<span class='el-icon el-icon-status-active' style='margin-right: 10px;'></span><%=rb.getString("JiHuo")%>
											</div>
											<div class='inactiveStatusItem' v-show="scope.row.op_state == '0'">
												<span class='el-icon el-icon-status-active' style='margin-right: 10px;'></span><%= rb.getString("QuJiHuo")%>
											</div>
										</template>
									</el-table-column>
									<el-table-column prop='product' min-width="120" label='<%=rb.getString("ChanPinLeiXing")%>'></el-table-column>
									<el-table-column prop='group_name' min-width="120" label='<%=rb.getString("SheBeiZu")%>'></el-table-column>
								</el-ctable>
								
							</el-tab-pane>
							<el-tab-pane name="cpe">
								<span slot="label" class="tabTitleCls">CPE</span>
								<el-ctable
								 	id="betaCpeTable"
									:url="betaCpeTableUrl"
									:query-params="betaCpeQueryParams" 
									ref="betaCpeTable" 
									height="100%"
									time=6
									row-key="macaddress"
									@selection-change='betaCpeSelectChange'
									:page-size="pageSize" 
									:page-list="pageList" 
									pagination="true">
										<!-- 列表toolbar -->
									<template slot="toolbar">
										<div style="display:flex;">
											<div class="productTypeBox">
												<el-select v-model="betaCpeQueryParams.product" size="mini" placeholder="<%=rb.getString("ChanPinXingHao")%>" filterable>
													<el-option v-for="item in betaCpeProductList" :label="item.text" :value="item.id">
													</el-option>
												</el-select>
											</div>
											<el-query type="normal" @query="queryBetaCpeTable" placeholder="<%=rb.getString("CPEBianMa")%> / <%=rb.getString("CPEName")%> / IMSI / MAC"></el-query>
										</div>
										
									</template>
										<!-- 列表columns -->
									<el-table-column type='selection' width="45" :selectable="checkSelectTable" :reserve-selection="true"></el-table-column>
									<el-table-column prop="connection_status" width="45s">
										<template slot-scope="scope">
											<div :class="{
												'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
												'':scope.row.have_connected==2,
												'conn_exc':scope.row.connection_status=='Exception',
												'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
										</template>
									</el-table-column>
									<el-table-column prop='serial_number' min-width="140" label='<%=rb.getString("CPEBianMa")%>'></el-table-column>
									<el-table-column prop='cpe_name' min-width="100" label='<%=rb.getString("CPEName")%>'></el-table-column>
									<el-table-column prop='imsi' min-width="60" label='<%=rb.getString("IMSI")%>'></el-table-column>
									<el-table-column prop='macaddress' min-width="80" label='MAC'></el-table-column>
									<el-table-column prop='model_name' min-width="120" label='<%=rb.getString("ChanPinXingHao")%>'></el-table-column>
									<el-table-column prop='group_name' min-width="120" label='<%=rb.getString("SheBeiZu")%>'></el-table-column>
									<el-table-column prop='host_name' min-width="120" label='<%=rb.getString("HostName")%>'></el-table-column>
								</el-ctable>
							</el-tab-pane>
						</el-tabs>
					</div>
					<div class="tableFooter">
						<div>
							<el-button type="primary" size="mini" @click="betaMigrationClick" :disabled="betaMigrationDisabled"><%=rb.getString("YiDongDaoShangYong")%></el-button>
							<el-button size="mini" @click="betaMigrationCancel"><%=rb.getString("QuXiao")%></el-button>
						</div>
					</div>
				</div>
			</div>
		</div>
		<div class="migrationConfirmBox" v-show="migrationConfirmShow">
			<div class="titleCls"><%=rb.getString("SheBeiQianYi")%></div>
			<div class="circleButton placeholder-bt" placeholder="<%=rb.getString("TuiChu")%>"  style="position: absolute;top:0px;right:0px;">		
				<span class="el-icon el-icon-circle-goback" @click="goBackClick"></span>
			</div>
			<div class="confirmTableBox">
				<div class="confirmPrompt" v-show="migrateType == 'CommercialToBeta' && !sasEnable" ><%=rb.getString("ShangYongQianYiTiShi")%></div>
				<div class="confirmPrompt" v-show="migrateType == 'BetaToCommercial' && !sasEnable"><%=rb.getString("BetaQianYiTiShi")%></div>
				<div class="confirmPrompt" v-show="sasEnable"><%=rb.getString("QianYiSASJingGao")%></div>
				<div class="confirmTableBoxMain">
					<el-tabs class="fit" v-model="confirmTableBoxActiveName">
						<el-tab-pane name="enb">
							<span slot="label" class="tabTitleCls">eNB</span>
							<div class="confirmTableInfoCls">
								<el-ctable 
									id="confirmBoxEnbSelectTable" 
									ref="confirmBoxEnbSelectTable" 
									:data="confirmBoxEnbSelectList" 
									:showHeader="false"
									:rownumber="true"
									front-pagination="true"
									height="100%" pagination="true" >
									<!-- 列表toolbar -->
									<template slot="toolbar">
										<div style="display:flex;margin-bottom:10px;position:relative;">
											<el-query ref="confirmBoxEnbSelectQuery" type="normal" @query="confirmBoxEnbSelectQuery" placeholder="<%=rb.getString("XiaoZhanBianMa")%>"></el-query>
											<div class="confirmBoxClearBtn" @click="clearConfirmBoxEnbSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
										</div>
									</template>
									<el-table-column prop="id" v-if="false"></el-table-column>
									<el-table-column prop="serial_number" label="<%=rb.getString("XiaoZhanBianMa")%>" min-width="160"></el-table-column>
									<el-table-column prop="host_name" label="<%=rb.getString("HostName")%>" min-width="160"></el-table-column>
									<el-table-column prop="product" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" min-width="160"></el-table-column>
									<el-table-column  label="<%=rb.getString("SheBeiZu")%>" min-width="200">
										<template slot-scope="scope" >
											<div class="tableItemCls">
												<span>{{scope.row.group_name}}</span>
												<span @click="delConfirmBoxEnbSelected(scope.row)" class="el-icon el-icon-circle-close icon_show"></span>
											</div>
										</template>
									</el-table-column>
								</el-ctable>
							</div>
						</el-tab-pane>
						<el-tab-pane name="cpe">
							<span slot="label" class="tabTitleCls">CPE</span>
							<div class="confirmTableInfoCls">
								<el-ctable 
									id="confirmBoxCpeSelectTable" 
									ref="confirmBoxCpeSelectTable" 
									:data="confirmBoxCpeSelectList" 
									:showHeader="false"
									:rownumber="true"
									front-pagination="true"
									height="100%" pagination="true" >
									<!-- 列表toolbar -->
									<template slot="toolbar">
										<div style="display:flex;margin-bottom:10px;position:relative;">
											<el-query ref="confirmBoxCpeSelectQuery"  type="normal" @query="confirmBoxCpeSelectQuery" placeholder="MAC"></el-query>
											<div class="confirmBoxClearBtn" @click="clearConfirmBoxCpeSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
										</div>
									</template>
									<el-table-column prop="id" v-if="false"></el-table-column>
									<el-table-column prop='serial_number' min-width="140" label='<%=rb.getString("CPEBianMa")%>'></el-table-column>
									<el-table-column prop='cpe_name' min-width="100" label='<%=rb.getString("CPEName")%>'></el-table-column>
									<el-table-column prop='imsi' min-width="60" label='<%=rb.getString("IMSI")%>'></el-table-column>
									<el-table-column prop='macaddress' min-width="80" label='MAC'></el-table-column>
									<el-table-column prop='model_name' min-width="120" label='<%=rb.getString("ChanPinXingHao")%>' width='122'></el-table-column>
									<el-table-column  min-width="200" label="<%=rb.getString("SheBeiZu")%>">
										<template slot-scope="scope" >
											<div class="tableItemCls">
												<span>{{scope.row.group_name}}</span>
												<span @click="delConfirmBoxCpeSelected(scope.row)" class="el-icon el-icon-circle-close icon_show"></span>
											</div>
										</template>
									</el-table-column>
								</el-ctable>
							</div>
						</el-tab-pane>
					</el-tabs>
				</div>
				<div class="tableFooter">
					<div style="padding-top:12px;">
						<el-button type="primary" size="mini" @click="deviceMigrationSubmit"><%=rb.getString("QueDing")%></el-button>
						<el-button size="mini" @click="goBackClick" ><%=rb.getString("QuXiao")%></el-button>
					</div>
				</div>
				
			</div>
		</div>
		<div class="migrationResultBox" v-show="migrationResultShow">
			<div class="titleCls"><%=rb.getString("SheBeiQianYiZhiXingJieGuo")%></div>
			<div class="circleButton placeholder-bt" placeholder="Device Move"  style="position: absolute;top:0px;right:0px;z-index:200">		
				<span class="el-icon el-icon-circle-move" @click="goDeviceMove"></span>
			</div>
			<div class="resultTableBox">
				<div class="resultTableBoxMain">
					
					<el-tabs class="fit" v-model="resultTableBoxActiveName" @tab-click="resultCloudTabChange">
						<el-tab-pane name='commercialCloud' label="Commercial Cloud" ></el-tab-pane>
						<el-tab-pane name='betaCloud' label="Beta Cloud" ></el-tab-pane>
					</el-tabs>
					<div class="resultTableBoxInfo">
						<div class="resultEnbAndCpeTabs">
							<div :class="resultEnbAndCpeShow == 'enb' ? 'tabEnbSelectCls' : 'tabEnbDefaultCls'" @click="resultEnbAndCpeTabChange('enb')" >
								<span class="el-icon el-icon-menu-eNB" style="margin-right:10px;"></span>eNB
							</div>
							<div :class="resultEnbAndCpeShow == 'cpe' ? 'tabcpeSelectCls' : 'tabcpeDefaultCls'" @click="resultEnbAndCpeTabChange('cpe')" >
								<span class="el-icon el-icon-menu-CPE" style="margin-right:10px;"></span>CPE
							</div>
						</div>
						
							<div class="placeholder-bt tabHeaderBtn" placeholder="<%=rb.getString("DaoChu")%>">
								<span class="el-icon el-icon-circle-export" @click="exportResultTable"></span>
							</div>
						
						<div class="enbListTable" v-show="resultEnbAndCpeShow == 'enb'">
							<div class="resultHeadBox">
								<div class="statisticsSuccessDiv">
									<div><span class="el-icon el-icon-circle-success" style="margin-right:5px;"></span><%=rb.getString("ChengGong")%></div>
									<div>{{statisticsSuccessEnb}}</div>
								</div>
								<div class="statisticsFailDiv">
									<div><span class="el-icon el-icon-circle-close" style="margin-right:5px;"></span><%=rb.getString("ShiBai")%></div>
									<div>{{statisticsFailEnb}}</div>
								</div>
							</div>
							<!--time=6 -->
							<el-ctable 
								ref="resultEnbTable" 
								id="resultEnbTable" 
								time=6
								:url="resultEnbTableUrl" 
								height="100%"
								:page-size="pageSize" 
								:page-list="pageList" 
								:query-params="queryResultEnbParams"
								@load-success="resultEnbTableLoadSuccess" 
								pagination="true"
								>
								<template slot="toolbar">
									<el-query type="normal" @query="queryResultEnb" placeholder='<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%>' style="padding-left:240px;"></el-query>
								</template>
								
								<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="200"  prop="serial_number" show-overflow-tooltip="true"></el-table-column>
								<el-table-column label='<%=rb.getString("HostName")%>' min-width="130" prop="host_name"></el-table-column>
								<el-table-column label='<%=rb.getString("ChanPinLeiXing")%>' min-width="130" prop="product"></el-table-column>
								<el-table-column label='<%=rb.getString("ZhuangTai")%>' min-width="120" prop="status">
									<template slot-scope="scope">
										<div class='statusItemInProgress' v-show="scope.row.status == 'inProgress'">
											<span class='el-icon el-icon-status-inProgress' style='margin-right: 10px;'></span><%=rb.getString("JinXingZhong")%>
										</div>
										<div class='statusItemSuccess' v-show="scope.row.status == 'success'">
											<span class='el-icon el-icon-circle-success' style='margin-right: 10px;'></span><%= rb.getString("ChengGong")%>
										</div>
										<div class='statusItemFail' v-show="scope.row.status == 'fail'">
											<span class='el-icon el-icon-circle-close' style='margin-right: 10px;'></span><%=rb.getString("ShiBai")%>
										</div>
									</template>
								</el-table-column>
								<el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' min-width="200" prop="failureReason"></el-table-column>
								<el-table-column label='<%=rb.getString("KaiShiShiJian")%>' min-width="150" prop="startTime"></el-table-column>
								<el-table-column label='<%=rb.getString("JieShuShiJian")%>' prop="endTime" min-width="150"></el-table-column>
							</el-ctable>
						</div>
						<div class="cpeListTable" v-show="resultEnbAndCpeShow == 'cpe'">
							<div class="resultHeadBox">
								<div class="statisticsSuccessDiv">
									<div><span class="el-icon el-icon-circle-success" style="margin-right:5px;"></span><%=rb.getString("ChengGong")%></div>
									<div>{{statisticsSuccessCpe}}</div>
								</div>
								<div class="statisticsFailDiv">
									<div><span class="el-icon el-icon-circle-close" style="margin-right:5px;"></span><%=rb.getString("ShiBai")%></div>
									<div>{{statisticsFailCpe}}</div>
								</div>
							</div>
							<!--time=6 -->
							<el-ctable 
								id="resultCpeTable"
								ref="resultCpeTable" 
								:url="resultCpeTableUrl" 
								:query-params="queryResultCpeParams"
								@load-success="resultCpeTableLoadSuccess" 
								height="100%"
								time=6
								:page-size="pageSize" :page-list="pageList"
								:pagination=true
							 >
								
								<template slot="toolbar">
									<el-query type="normal" @query="queryResultCpe" placeholder='<%=rb.getString("CPEBianMa")%> / <%=rb.getString("CPEName")%> / IMSI' style="margin-left:240px;"></el-query>
								</template>
								<el-table-column prop='serial_number' label='<%=rb.getString("CPEBianMa")%>' min-width="200"></el-table-column>
								<el-table-column prop='cpe_name' label='<%=rb.getString("CPEName")%>' min-width="100"></el-table-column>
								<el-table-column prop='imsi' label='<%=rb.getString("IMSI")%>' min-width="140"></el-table-column>
								<el-table-column prop='macaddress' label='MAC' min-width="140"></el-table-column>
								<el-table-column prop='model_name' label='<%=rb.getString("ChanPinXingHao")%>' min-width="140"></el-table-column>
								<el-table-column label='<%=rb.getString("ZhuangTai")%>' min-width="120" prop="status">
									<template slot-scope="scope">
										<div class='statusItemInProgress' v-show="scope.row.status == 'inProgress'">
											<span class='el-icon el-icon-status-inProgress' style='margin-right: 10px;'></span><%=rb.getString("JinXingZhong")%>
										</div>
										<div class='statusItemSuccess' v-show="scope.row.status == 'success'">
											<span class='el-icon el-icon-circle-success' style='margin-right: 10px;'></span><%= rb.getString("ChengGong")%>
										</div>
										<div class='statusItemFail' v-show="scope.row.status == 'fail'">
											<span class='el-icon el-icon-circle-close' style='margin-right: 10px;'></span><%=rb.getString("ShiBai")%>
										</div>
									</template>
								</el-table-column>
								<el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>'  prop="failureReason" min-width="200"></el-table-column>
								<el-table-column label='<%=rb.getString("KaiShiShiJian")%>' prop="startTime" min-width="160"></el-table-column>
								<el-table-column label='<%=rb.getString("JieShuShiJian")%>' prop="endTime" min-width="160"></el-table-column>
							</el-ctable>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
	
</div>
	

<script type="text/javascript">
new Vue({
	el:'#deviceMigrationPage',
	data(){
		return{
			
			deviceMigrationMainShow:true,
			migrationConfirmShow:false,
			migrationResultShow:false,
			height:'370px',
			migrateType:'CommercialToBeta',
			sasEnable:false,
			pageSize:50,
			pageList:[50,100,200],
			commercialEnbQueryParams:{
				timeZone:timeZone,
				searchText:'',
				product:''
			},
			commercialCpeQueryParams:{
				timeZone:timeZone,
				searchText:'',
				product:''
			},
			betaEnbQueryParams:{
				timeZone:timeZone,
				searchText:'',
				product:'',
			},
			betaCpeQueryParams:{
				timeZone:timeZone,
				searchText:'',
				product:''
			},
			queryResultEnbParams:{
				timeZone:timeZone,
				searchText:'',
			},
			queryResultCpeParams:{
				timeZone:timeZone,
				searchText:'',
			},
			commercialEnbProductList:[],
			commercialCpeProductList:[],
			betaEnbProductList:[],
			betaCpeProductList:[],
		
			commercialEnbTableUrl:'${ctx}/migrate/commercialDevice/queryEnbList.action',
			commercialCpeTableUrl:'${ctx}/migrate/commercialDevice/queryCpeList.action',
			betaEnbTableUrl:'${ctx}/migrate/betaDevice/queryEnbList.action',
			betaCpeTableUrl:'${ctx}/migrate/betaDevice/queryCpeList.action',
			resultEnbAndCpeShow:'enb',
			resultEnbTableUrl:'${ctx}/migrate/commercialResult/queryEnbList.action',
			resultCpeTableUrl:'${ctx}/migrate/commercialResult/queryCpeList.action',
			commercialActiveName:'enb',
			betaActiveName:'enb',
			commercialSelectActiveName:'enb',
			betaSelectActiveName:'enb',
			confirmTableBoxActiveName:'enb',
			resultTableBoxActiveName:'commercialCloud',
            statisticsSuccessEnb:'',
            statisticsFailEnb:'',
            statisticsSuccessCpe:'',
            statisticsFailCpe:'',
			commercialEnbSelectList:[],
			commercialCpeSelectList:[],
			betaEnbSelectList:[],
			betaCpeSelectList:[],
			commercialEnbCodes:'',
			commercialCpeCodes:'',
			betaEnbCodes:'',
			betaCpeCodes:'',
			confirmBoxEnbSelectList:[],
			confirmBoxCpeSelectList:[],
			commercialSelectShow:false,
			betaSelectShow:false,
			enbTableList:[{serial_number:'serial_number_001',host_name:'name_001',connection_status:'',product:'RTS',op_state:'0',group_name:'Default Device Group'},{serial_number:'serial_number_002',host_name:'name_002',connection_status:'',product:'QAFA',op_state:'1',group_name:'Default Device Group'}],
			cpeTableList:[{macaddress:'9a:4f:74:03:1f:71',cpe_name:'name_001',connection_status:'',model_name:'BaiCE_AP_1.2'},{mac_address:'9a:4f:74:03:1f:72',cpe_name:'name_002',connection_status:'',model_name:'BaiCE_AP_1.2'}]
		}
	},
	computed:{
		commercialSelectNum:function(){
			return this.commercialEnbSelectList.length + this.commercialCpeSelectList.length
		},
		betaSelectNum:function(){
			return this.betaEnbSelectList.length + this.betaCpeSelectList.length
		},
		betaMigrationDisabled:function(){
			return this.betaEnbSelectList.length == 0 && this.betaCpeSelectList.length == 0
		},
		commercialMigrationDisabled:function(){
			return this.commercialEnbSelectList.length == 0 && this.commercialCpeSelectList.length == 0
		},
	},
	methods:{
		// 初始化
		init(){
			var vm = this;
			axios.post('${ctx}/migrate/commercialProduct/queryEnbProductList.action').then(function(response){
				var data = response.data;
				vm.commercialEnbProductList = data;
			}).catch(function(error){});
			axios.post('${ctx}/migrate/commercialProduct/queryCpeProductList.action').then(function(response){
				var data = response.data;
				vm.commercialCpeProductList = data;
			}).catch(function(error){});
			axios.post('${ctx}/migrate/betaProduct/queryEnbProductList.action').then(function(response){
				var data = response.data;
				vm.betaEnbProductList = data;
			}).catch(function(error){});
			axios.post('${ctx}/migrate/betaProduct/queryCpeProductList.action').then(function(response){
				var data = response.data;
				vm.betaCpeProductList = data;
			}).catch(function(error){});
		},
		// 不在线的设备不可选
		checkSelectTable(row,index){
			return row.connection_status !== 'Off';
		},
		// 打开商用环境已选弹窗
		openCommercialSelectTable(){
			var vm = this;
			//打开已选弹窗时，默认选中'enb''
			vm.commercialSelectActiveName = 'enb';
			vm.commercialSelectShow = true
		},
		// 关闭商用已选弹窗
		closeCommercialSelectTable(){
			var vm = this;
			vm.commercialSelectShow = false;
		},
		// 打开Beta环境已选弹窗
		openBetaSelectTable(){
			var vm = this;
			//打开已选弹窗时，默认选中'enb''
			vm.betaSelectActiveName = 'enb';
			vm.betaSelectShow = true
		},
		// 关闭Beta环境已选弹窗
		closeBetaSelectTable(){
			var vm = this;
			vm.betaSelectShow = false;
		},
		// 商用环境 enb设备表格搜索
		queryCommercialEnbTable(val){
			var vm = this;
            vm.commercialEnbQueryParams.searchText = val;
		},
		// 商用环境 cpe设备表格搜索
		queryCommercialCpeTable(val){
			var vm = this;
            vm.commercialCpeQueryParams.searchText = val;
		},
		// beta环境 enb设备表格搜索
		queryBetaEnbTable(val){
			var vm = this;
            vm.betaEnbQueryParams.searchText = val;
		},
		// beta环境 cpe设备表格搜索
		queryBetaCpeTable(val){
			var vm = this;
            vm.betaCpeQueryParams.searchText = val;
		},
		// 商用环境 enb设备表格选择事件
		commercialEnbSelectChange(selection){
			var vm = this,selectEnbData=[],cellCodeList=[];

			selectEnbData = selection ? selection : [];
			selectEnbData.map((item)=>{
				item.snAndCellName = item.serial_number + '\xa0\xa0\xa0' + item.host_name;
				cellCodeList.push(item.small_cell_code)
			})
			vm.commercialEnbSelectList = selectEnbData;
			vm.commercialEnbCodes = cellCodeList.join(',');
			
		},
		// 商用环境 cpe设备表格选择事件
		commercialCpeSelectChange(selection){
			var vm = this,selectCpeData=[],cellCodeList=[];

			selectCpeData = selection ? selection : [];
			selectCpeData.map((item)=>{
				item.macAndCPEName = item.macaddress + '\xa0\xa0\xa0' + item.cpe_name;
				cellCodeList.push(item.cpe_code)
			})
			vm.commercialCpeSelectList = selectCpeData;
			vm.commercialCpeCodes = cellCodeList.join(',');
		},
		// beta环境 enb设备表格选择事件
		betaEnbSelectChange(selection){
			var vm = this,selectEnbData=[],cellCodeList=[];
			selectEnbData = selection ? selection : [];
			selectEnbData.map((item)=>{
				item.snAndCellName = item.serial_number + '\xa0\xa0\xa0' + item.host_name;
				cellCodeList.push(item.small_cell_code)
			})
			vm.betaEnbSelectList = selectEnbData;
			vm.betaEnbCodes = cellCodeList.join(',');
		},
		// beta环境 cpe设备表格选择事件
		betaCpeSelectChange(selection){
			var vm = this,selectCpeData=[],cellCodeList=[];

			selectCpeData = selection ? selection : [];
			selectCpeData.map((item)=>{
				item.macAndCPEName = item.macaddress + '\xa0\xa0\xa0' + item.cpe_name;
				cellCodeList.push(item.cpe_code)
			})
			vm.betaCpeSelectList = selectCpeData;
			vm.betaCpeCodes = cellCodeList.join(',');
		},
		// 商用环境 enb设备已选表格 清空事件
		clearCommercialEnbSelected(){
			var vm = this;

			vm.$refs["commercialEnbTable"].clearSelection();
		},
		// 商用环境 cpe设备已选表格 清空事件
		clearCommercialCpeSelected(){
			var vm = this;

			vm.$refs["commercialCpeTable"].clearSelection();
		},
		// beta环境 enb设备已选表格 清空事件
		clearBetaEnbSelected(){
			var vm = this;

			vm.$refs["betaEnbTable"].clearSelection();
		},
		// beta环境 cpe设备已选表格 清空事件
		clearBetaCpeSelected(){
			var vm = this;

			vm.$refs["betaCpeTable"].clearSelection();
		},
		// 商用环境 enb设备已选表格 单个删除事件
		delCommercialEnbSelected(rows){
			var vm = this;
			vm.$refs["commercialEnbTable"].toggleRowSelection(rows,false);
		},
		// 商用环境 cpe设备已选表格 单个删除事件
		delCommercialCpeSelected(rows){
			var vm = this;
			vm.$refs["commercialCpeTable"].toggleRowSelection(rows,false);
		},
		// beta环境 enb设备已选表格 单个删除事件
		delBetaEnbSelected(rows){
			var vm = this;
			vm.$refs["betaEnbTable"].toggleRowSelection(rows,false);
		},
		// beta环境 cpe设备已选表格 单个删除事件
		delBetaCpeSelected(rows){
			var vm = this;
			vm.$refs["betaCpeTable"].toggleRowSelection(rows,false);
		},
		// 商用环境 设备迁移按钮
		commercialMigrationClick(){
			var vm = this,allData=[];
			vm.sasEnable = false;
			vm.confirmTableBoxActiveName = 'enb';
			vm.migrateType = 'CommercialToBeta';
			vm.confirmBoxEnbSelectList = vm.commercialEnbSelectList;
			vm.confirmBoxCpeSelectList = vm.commercialCpeSelectList;
			allData = vm.commercialEnbSelectList.concat(vm.commercialCpeSelectList);
			allData.map((item)=>{
				if(item.sasEnable == 'on'){
					vm.sasEnable = true;
				}
			})
			vm.deviceMigrationMainShow = false;
			vm.migrationConfirmShow = true;
			vm.migrationResultShow = false;
			

		},
		// beta环境 设备迁移按钮
		betaMigrationClick(){
			var vm = this,allData=[];
			vm.sasEnable = false;
			vm.confirmTableBoxActiveName = 'enb';
			vm.migrateType = 'BetaToCommercial';
			vm.confirmBoxEnbSelectList = vm.betaEnbSelectList;
			vm.confirmBoxCpeSelectList = vm.betaCpeSelectList;
			allData = vm.betaEnbSelectList.concat(vm.betaCpeSelectList);
			allData.map((item)=>{
				if(item.sasEnable == 'on'){
					vm.sasEnable = true;
				}
			})
			vm.deviceMigrationMainShow = false;
			vm.migrationConfirmShow = true;
			vm.migrationResultShow = false;
			
		},
		// 商用环境 设备迁移取消按钮
		commercialMigrationCancel(){
			var vm = this;
			vm.$refs["commercialEnbTable"].clearSelection();
			vm.$refs["commercialCpeTable"].clearSelection();
		},
		// 商用环境 设备迁移取消按钮
		betaMigrationCancel(){
			var vm = this;
			vm.$refs["betaEnbTable"].clearSelection();
			vm.$refs["betaCpeTable"].clearSelection();
		},
		// 确认页面 enb表格搜索
		confirmBoxEnbSelectQuery(val){
			var vm = this;
			if(val){
				if(vm.migrateType == 'CommercialToBeta'){
					var data = vm.commercialEnbSelectList.filter((item)=>{
						return item.serial_number.indexOf(val) != -1
					})
					vm.confirmBoxEnbSelectList = data;
				}else{
					var data = vm.betaEnbSelectList.filter((item)=>{
						return item.serial_number.indexOf(val) != -1
					})
					vm.confirmBoxEnbSelectList = data;
				}
			}else{
				if(vm.migrateType == 'CommercialToBeta'){
					vm.confirmBoxEnbSelectList = vm.commercialEnbSelectList;
				}else{
					vm.confirmBoxEnbSelectList = vm.betaEnbSelectList;
				}
			}
		},
		// 确认页面 cpe表格搜索
		confirmBoxCpeSelectQuery(val){
			var vm = this;
			if(val){
				if(vm.migrateType == 'CommercialToBeta'){
					var data = vm.commercialCpeSelectList.filter((item)=>{
						return item.serial_number.indexOf(val) != -1
					})
					vm.confirmBoxCpeSelectList = data;
				}else{
					var data = vm.betaCpeSelectList.filter((item)=>{
						return item.serial_number.indexOf(val) != -1
					})
					vm.confirmBoxCpeSelectList = data;
				}
			}else{
				if(vm.migrateType == 'CommercialToBeta'){
					vm.confirmBoxCpeSelectList = vm.commercialCpeSelectList;
				}else{
					vm.confirmBoxCpeSelectList = vm.betaCpeSelectList;
				}
			}
			
		},
		// 确认页面 enb 清空事件
		clearConfirmBoxEnbSelected(){
			var vm = this;
			vm.confirmBoxEnbSelectList = [];
			if(vm.migrateType == 'CommercialToBeta'){
				vm.$refs["commercialEnbTable"].clearSelection();
			}else{
				vm.$refs["betaEnbTable"].clearSelection();
			}
		},
		// 确认页面 cpe 清空事件
		clearConfirmBoxCpeSelected(){
			var vm = this;
			vm.confirmBoxCpeSelectList = [];
			if(vm.migrateType == 'CommercialToBeta'){
				vm.$refs["commercialCpeTable"].clearSelection();
			}else{
				vm.$refs["betaCpeTable"].clearSelection();
			}
		},
		// 确认页面 enb 单个删除事件
		delConfirmBoxEnbSelected(rows){
			var vm = this;
			vm.confirmBoxEnbSelectList.splice(vm.confirmBoxEnbSelectList.findIndex(item => item.serial_number === rows.serial_number ),1);
			if(vm.migrateType == 'CommercialToBeta'){
				vm.$refs["commercialEnbTable"].toggleRowSelection(rows,false);
			}else{
				vm.$refs["betaEnbTable"].toggleRowSelection(rows,false);
			}
			
		},
		// 确认页面 cpe 单个删除事件
		delConfirmBoxCpeSelected(rows){
			var vm = this;
			vm.confirmBoxCpeSelectList.splice(vm.confirmBoxCpeSelectList.findIndex(item => item.serial_number === rows.serial_number ),1);
			if(vm.migrateType == 'CommercialToBeta'){
				vm.$refs["commercialCpeTable"].toggleRowSelection(rows,false);
			}else{
				vm.$refs["betaCpeTable"].toggleRowSelection(rows,false);
			}
		},
		// 确认页面 返回主页面
		goBackClick(){
			var vm = this;
			vm.commercialActiveName = 'enb';
			vm.betaActiveName = 'enb';
			vm.$refs["confirmBoxEnbSelectQuery"].reset();
			vm.$refs["confirmBoxCpeSelectQuery"].reset();
			vm.deviceMigrationMainShow = true;
			vm.migrationConfirmShow = false;
			vm.migrationResultShow = false;
		},
		// 查看结果页面
		goResult(){
			var vm = this;
			vm.resultTableBoxActiveName = 'commercialCloud';
			vm.resultEnbAndCpeShow = 'enb';
			vm.deviceMigrationMainShow = false;
			vm.migrationConfirmShow = false;
			vm.migrationResultShow = true;
		},
		// 结果页面 返回主页面
		goDeviceMove(){
			var vm = this;
			vm.commercialActiveName = 'enb';
			vm.betaActiveName = 'enb';
			vm.deviceMigrationMainShow = true;
			vm.migrationConfirmShow = false;
			vm.migrationResultShow = false;
		},
		// 结果导出
		exportResultTable(){
            var vm = this,urls='',
				params={
            		timeZone:timeZone,
					searchText:'',
				};
            if(vm.resultTableBoxActiveName == 'commercialCloud'){
				if(vm.resultEnbAndCpeShow == 'enb'){
					urls = '${ctx}/migrate/commercialResult/exportEnbList.action';
					params.searchText = vm.queryResultEnbParams.searchText;
				}else{
					urls = '${ctx}/migrate/commercialResult/exportCpeList.action';
					params.searchText = vm.queryResultCpeParams.searchText;
				}
			}else{
				if(vm.resultEnbAndCpeShow == 'enb'){
					urls = '${ctx}/migrate/betaResult/exportEnbList.action';
					params.searchText = vm.queryResultEnbParams.searchText;
				}else{
					urls = '${ctx}/migrate/betaResult/exportCpeList.action';
					params.searchText = vm.queryResultCpeParams.searchText;
				}
			}
			exportByForm(urls,params);
		},
		// 结果页面 enb 搜索事件
		queryResultEnb(val){
            var vm = this;
            vm.queryResultEnbParams.searchText = val;
		},
		// 结果页面 cpe 搜索事件
		queryResultCpe(val){
            var vm = this;
            vm.queryResultCpeParams.searchText = val;
		},
		// 结果页面 enb表格 加载成功回调
		resultEnbTableLoadSuccess(data){
            var vm = this;
			vm.statisticsSuccessEnb = data.properties.successCount;
			vm.statisticsFailEnb = data.properties.failCount;
		},
		// 结果页面 cpe表格 加载成功回调
		resultCpeTableLoadSuccess(data){
            var vm = this;
			vm.statisticsSuccessCpe = data.properties.successCount;
			vm.statisticsFailCpe = data.properties.failCount;
		},
		// 结果页面 cpe enb 切换事件
		resultEnbAndCpeTabChange(val){
			var vm = this;
			
			if(vm.resultEnbAndCpeShow == val)return
			vm.resultEnbAndCpeShow = val

		},
		// 结果页面  商用环境和Beta环境切换
		resultCloudTabChange(){
			var vm = this;
			vm.resultEnbAndCpeShow = 'enb';
			if(vm.resultTableBoxActiveName == 'commercialCloud'){
				vm.resultEnbTableUrl = '${ctx}/migrate/commercialResult/queryEnbList.action';
				vm.resultCpeTableUrl = '${ctx}/migrate/commercialResult/queryCpeList.action';
			}else{
				vm.resultEnbTableUrl = '${ctx}/migrate/betaResult/queryEnbList.action';
				vm.resultCpeTableUrl = '${ctx}/migrate/betaResult/queryCpeList.action';
			}
			
		},
		// 设备迁移提交
		deviceMigrationSubmit(){
			var vm = this,
				urls='${ctx}/migrate/execute.action'
				params={
					direction:vm.migrateType,
					enbCodeList:'',
					cpeCodeList:'',
				};
			if(vm.migrateType == 'CommercialToBeta'){
				params.enbCodeList = vm.commercialEnbCodes;
				params.cpeCodeList = vm.commercialCpeCodes;
			}else{
				params.enbCodeList = vm.betaEnbCodes;
				params.cpeCodeList = vm.betaCpeCodes;
			}
			axios.post(urls,stringify(params)).then(function(response){
				var data = response.data;
				if(data) {
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success'
						});
						vm.$refs["commercialEnbTable"].refresh();
						vm.$refs["commercialCpeTable"].refresh();
						vm.$refs["betaEnbTable"].refresh();
						vm.$refs["betaCpeTable"].refresh();
						if(vm.migrateType == 'CommercialToBeta'){
							vm.commercialMigrationCancel();
						}else{
							vm.betaMigrationCancel();
						}
						vm.goBackClick();
						
					}else{
						vm.$message.error(data["message"])
					}
				}
			}).catch(function(error){});
		},
	},
	mounted(){
		this.init();
	}

})
</script>