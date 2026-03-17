<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
	#gnbBackupRestoreTask .el-form-item__label{
		line-height:26px;
		width:160px;
		text-align:left;
	}
	#gnbBackupRestoreTask .el-select .el-input.is-disabled .el-input__inner{
		min-height:26px;
		max-height:26px;
	}
	#gnbBackupRestoreTask .newBackupTaskWarp .basicInfoWarp,
	#gnbBackupRestoreTask .newBackupTaskWarp .selectDeviceWarp{
		border-bottom:1px solid #E9E9E9;
	}
	#gnbBackupRestoreTask .newBackupTaskWarp .basicInfoWarp .basicInfo{
		padding:40px 50px 20px;
	}
	#gnbBackupRestoreTask .newBackupTaskWarp .selectDeviceWarp .selectDeviceInfo{
		padding:30px 50px 20px;
	}
	#gnbBackupRestoreTask .newBackupTaskWarp .newTaskName{
		margin-left:76px;
	}
	#gnbBackupRestoreTask .newBackupTaskWarp .basicInfoWarp .newTaskName .el-input{
		width:348px;
		height:26px;
		line-height:26px;
	}
	#gnbBackupRestoreTask .newBackupTaskWarp .basicInfoWarp .newTaskName .el-form-item__error{
		margin-left:160px;
	}
	#gnbBackupRestoreTask .selectDeviceBox{
		margin: 0 72px;
		background:#FFFFFF;
		display:flex;
		height:300px;
	}
	#gnbBackupRestoreTask .selectDeviceBox .el-radio{
		margin-right:30px;
	}
	#gnbBackupRestoreTask .leftDeviceSpecific {
		width:260px;
		height:298px;
		border:1px solid #E9E9E9;
	}
	#gnbBackupRestoreTask .leftDeviceSpecific p{
		height:36px;
		line-height:36px;
		text-align:center;
		background:#F6F7FB;
		border-bottom:1px solid #E9E9E9;
	}
	#gnbBackupRestoreTask .leftDeviceSpecific .el-radio__label{
		font-size:12px;
	}
	#gnbBackupRestoreTask .executeModeBox{
		padding: 20px 0 0 76px;
		display:flex;
	}

	#gnbBackupRestoreTask .timeSelect{
		display:inline-block;
		vertical-align:bottom;
		margin-left:15px;
		margin-bottom:25px;
		margin-top:-4px;
	}
	#gnbBackupRestoreTask .closeSlide{
		position:absolute;
		right:30px;
		top:20px;
	}
	#gnbBackupRestoreTask .el-pairgrid .el-pairgrid-title{
		display: none;
	}
	#gnbBackupRestoreTask .el-pairgrid .el-pairgrid-title:last-child{
        top: 10px;
        right: 10px !important;
	}
	#gnbBackupRestoreTask .pairgrid-left .el-query{
		margin-left:0 !important;
	}
	#gnbBackupRestoreTask .pairgrid-right{
		top:38px !important;
		border:1px solid #E9E9E9;
	}
	#gnbBackupRestoreTask .noSelectDevice .el-form-item__error{
		/*left:50px;*/
		margin-left: 76px;
	}
	#gnbBackupRestoreTask .el-radio{
		font-weight:unset;
	}
	#gnbBackupRestoreTask .el-icon-time{
		font-size:14px;
	}
	#gnbBackupRestoreTask .gnbBackupSearchCon{
		margin-left:18px;
	}
	#gnbBackupRestoreTask .gnbBackupSearchCon .el-input--suffix{
		width:460px;
		height:28px;
	}
	#gnbBackupRestoreTask .gnbBackupSearchCon .el-input--suffix .el-input__inner{
		height:28px;
		line-height:28px;
		padding-left:15px;
		padding-right:40px;
		margin-top: 1px;
	}
	#gnbBackupRestoreTask .gnbBackupSearchCon .el-icon-common-search{
		margin-top:5px;
		margin-right:10px;
	}
	#gnbBackupRestoreTask .boxBorderCon .el-radio-group{
		margin-top:6px;
	}
	#gnbBackupRestoreTask .infoTip{
		font-size:14px;
		margin-right:5px;
	}
	#gnbBackupRestoreTask .infoTip:before{
		color:#4D84FF;
	}
	#gnbBackupRestoreTask .el-radio.is-bordered { 
		max-width: 160px; 
		height: 30px; 
		padding: 7px 12px; 
	}
	#gnbBackupRestoreTask .editButton{
		position: absolute;
		right: 130px;
		top: -5px;
		padding:0 10px;
		height:24px;
		background:#F2F9FF;
		border-radius:2px;
		line-height:24px;
		cursor:pointer;
		margin-left:10px;
		border:1px solid #1DA3FC;
		display:inline-block;
	}
	#gnbBackupRestoreTask .editButton i{
		font-size:14px !important;
	}
	#gnbBackupRestoreTask .editButton span{
		font-size:12px;
	}
	#gnbBackupRestoreTask .inputCommon .el-input__inner { 
		border-radius: 4px; 
	}
	#gnbBackupRestoreTask .validateItem .el-input-group__append { 
		border:none; 
		background:none; 
		padding: 0px 10px; 
	}
	#gnbBackupRestoreTask .validateItem .el-input__inner { 
		width:186px; 
	}
	#gnbBackupRestoreTask .is-error .el-input-group__append{ 
		color:#FA5555; 
	}
	#gnbBackupRestoreTask .validateItem .el-input-group__append { 
		border:none; background:none; 
	}
	#gnbBackupRestoreTask .validateItem .el-form-item__error { 
		display:none; 
	}
	#gnbBackupRestoreTask .el-form-item__error {
		margin-left: 0;
	}
	/* Time 字段上下结构样式 */
	#gnbBackupRestoreTask .timeFieldTop .el-form-item__label {
		width: auto !important;
		text-align: left !important;
		margin-bottom: 5px;
		display: block;
	}
	#gnbBackupRestoreTask .timeFieldTop .el-form-item__content {
		margin-left: 0 !important;
	}
	.selectDateItem {
		font-size: 12px;
		font-weight: normal;
		transition: all 0.3s ease;
		text-align: center;
		cursor: pointer;
		border-radius: 50%;
	}
	#gnbBackupRestoreTask .commonWidthItem .el-input {
		width: 300px;
	}
	/* FTP Server 样式 */
	#gnbBackupRestoreTask .executeModeWarp {
		padding-bottom: 50px;
	}
	#gnbBackupRestoreTask .executeModeWarp .executeModeInfor {
		padding: 30px 50px 0;
	}
	#gnbBackupRestoreTask .ftpServerContainer {
		padding: 20px 0 0 76px;
		display: flex;
		flex-wrap: wrap;
		gap: 0 180px;
		max-width: 800px;
	}
	#gnbBackupRestoreTask .ftpServerContainer .el-form-item {
		flex: 0 0 300px;
	}
	#gnbBackupRestoreTask .ftpServerContainer .el-input,
	#gnbBackupRestoreTask .ftpServerContainer .el-password {
		width: 300px;
	}
	#gnbBackupRestoreTask .ftpServerContainer .el-input[style*="display: none"] {
		display: none !important;
	}
	
	/* 执行模式信息标题样式 */
	#gnbBackupRestoreTask .executeModeInfor.group-title {
		padding: 30px 50px 0;
	}
	/* 定时执行单选按钮样式 */
	#gnbBackupRestoreTask .executeModeBox .el-radio[label="timing"] {
		margin-bottom: 0px;
		margin-left: 80px;
	}
	/* Execute 标签必填星号样式 */
	#gnbBackupRestoreTask .execute-label-required {
		color: #FF4614;
	}
	
	/* Week 选择器 Popover 内容 */
	#gnbBackupRestoreTask .week-popover-content {
		padding: 10px 20px;
	}
	#gnbBackupRestoreTask .week-days-grid {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		gap: 8px;
	}
	/* Month 选择器 Popover 内容 */
	#gnbBackupRestoreTask .month-popover-content {
		padding: 15px;
		background: #F9FAFC;
		border-radius: 8px;
	}
	#gnbBackupRestoreTask .month-popover-title {
		margin-bottom: 10px;
		font-size: 13px;
		color: #606266;
		font-weight: bold;
	}
	#gnbBackupRestoreTask .month-days-grid {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		gap: 10px 20px;
	}
	/* 选择器输入框指针样式 */
	#gnbBackupRestoreTask .selector-input-readonly {
		cursor: pointer;
	}
</style>

<!-- gNB-备份与恢复 -新建，修改，查看 备份，恢复及周期备份任务 -->
<div id="gnbBackupRestoreTask">
	<div class="newBackupTaskWarp">
		<el-form :model='gnbRuleForm' :rules="gnbRules" ref="gnbRuleForm">
			<!-- 基本信息 -->
			<div class="basicInfoWarp" v-if="gnbAddTaskType == 'backup' || gnbAddTaskType == 'restore'">
				<div class="group-title not-extend basicInfo">
					<span class="title-icon"></span>
					<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
				</div>
				<el-form-item class="newTaskName" label='<%=rb.getString("RenWuMingCheng")%>' prop='taskName'>
					<el-input maxlength="100" v-model="gnbRuleForm.taskName"></el-input>
				</el-form-item>
				<el-form-item class="newTaskName" label='<%=rb.getString("HuiFuLeiXing")%>' prop='gnbRestoreType' v-if="gnbAddTaskType == 'restore'">
					<div class="boxBorderCon">
						<el-radio-group v-model='gnbRestoreType'>
							<el-radio label="restore"><%=rb.getString("HuiFuDaoZuiXinGengXinDeWenJianPeiZhi")%></el-radio>
							<el-radio label="reset" style="margin-left:40px;"><%=rb.getString("HuiFuChuChangPeiZhi")%></el-radio>
						</el-radio-group>
					</div>
				</el-form-item>
			</div>
			<!--周期备份开关-->
			<div v-else>
				<el-form-item label='<%=rb.getString("SheZhiKaiGuan")%>' prop='enable' label-width='40px' class="executeModeBox" style="font-weight: bold;margin-bottom: 0; padding: 20px 0 0 54px;">
					<el-switch v-model="gnbRuleForm.enable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
				</el-form-item>
			</div>
			<!--设备选择  -->
			<div class="selectDeviceWarp">
				<div class="group-title not-extend selectDeviceInfo">
					<span class="title-icon"></span>
					<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
				</div>
                <el-form-item class="newTaskName" label='<%=rb.getString("SheBeiZhiDing")%>'>
                    <el-radio-group v-model="gnbDeviceType">
                        <el-radio label="1" border><%=rb.getString("QuanBu")%></el-radio>
                        <el-radio label="2" border><%=rb.getString("ZhiDingZhiXing")%></el-radio>
                    </el-radio-group>
                </el-form-item>
				<div class='selectDeviceBox'>
					<div style='width:100%;'>
						<div style='display:flex;height:300px;width:100%; '>
							<div style='flex:1'>
								<el-pairgrid v-show="gnbDeviceTypeSpecified"
									:id="'gnbBackupRestoreTaskTable'"
									:rownumber="true"
									ref="gnbBackupRestoreTaskTable"
									@selection-change='gnbSelectChange'
									:right-url="gnbRightUrl"
									:left-url="gnbLeftUrl" :height="gnbHeight"
									row-key="small_cell_code"
									:query-params="gnbDeviceSelectParams"
									:title="gnbDeviceTitle"
									:messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}">
									<template slot="left">
										<el-table-column type='selection' width="45"></el-table-column>
										<el-table-column prop="connection_status" width="50">
											<template slot-scope="scope">
												<div :class="{
													'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
													'':scope.row.have_connected==2,
													'conn_exc':scope.row.connection_status=='Exception',
													'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
											</template>
										</el-table-column>
										<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number" sortable></el-table-column>
										<el-table-column label='<%=rb.getString("HostName")%>' prop="host_name" sortable></el-table-column>
										<el-table-column label='<%=rb.getString("ChanPinLeiXing")%>' prop="product_type" sortable></el-table-column>

										<el-table-column label='<%=rb.getString("ZuiXinGengXinWenJian")%>' prop="latest_update_file"></el-table-column>
										<el-table-column label='<%=rb.getString("ZuiXianGengXinShiJian")%>' prop="latest_update_time" sortable></el-table-column>
									</template>
									<!-- 模糊查询 -->
									<template slot="toolbar">
					               		<div class="gnbBackupSearchCon commonFlex" style="position: relative;">
					               		    <span class="commonText14" style='line-height: 28px; margin-right: 20px;'><%=rb.getString("gNBSheBei")%></span>
											<el-input placeholder="<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>" suffic-icon='el-icon-search' v-model="gnbSpecifiedDevicesSearch">
												<i slot="suffix" class="el-icon el-icon-common-search"  @click="gnbDeviceSelectQuery"></i>
											</el-input>

											<div class="tableHeadQueryBoxCls" style='padding: 0 10px;'>
                                                <div v-for="(item,index) in advancedQueryItemList" style='margin: -5px 0 0;'>
													<div v-if="item.type == 'checkbox' && item.isShow" style="margin-right:10px;">
														<el-popfilter
															:label='item.label'
															v-model="item.checkedItemList"
															:list="item.options"
															:visible.sync="item.isShow"
															@check-change="advanceQuery(item.type,item.value,item.checkedItemList)">
														</el-popfilter>
													</div>
													<div v-if="item.type == 'select' && item.isShow" style="margin-right:10px;">
														<el-popfilter
															type="single"
															:label='item.label'
															v-model="item.selectVal"
															:list="item.options"
															:visible.sync="item.isShow"
															@check-change="advanceQuery(item.type,item.value,item.selectVal)">
														</el-popfilter>
													</div>
                                                </div>
                                                <div class="advancedQueryItemBox"  style="background: #FFF;" @click="clearFilterClick">
                                                    <%=rb.getString("QingKongShaiXuan")%>
                                                </div>
												<div class="editButton" @click="gnbBatchBtnClick" size="mini">
													<i class="el-icon el-icon-batchInput" style="font-size: 14px;padding-right: 5px;"></i>
													<span><%=rb.getString("PiLiangShuRu")%></span>
												</div>
                                            </div>
										</div>
					                </template>
									<!--右侧的下拉表格 -->
									<template slot='right'>
										<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
										<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>'></el-table-column>
									</template>
								</el-pairgrid>

								<el-ctable v-if="gnbDeviceTypeAll"
									ref="gnbBackupRestoreTaskTableAll"
									id="gnbBackupRestoreTaskTableAll"
									row-key="small_cell_code"
									:url="gnbDeviceUrl"
									:height="gnbHeight"
									:query-params="gnbParamsAll"
									pagination="true"
									style="border:1px solid #E9E9E9;height:300px;">
									<el-table-column prop="connection_status" width="50">
										<template slot-scope="scope">
											<div :class="{
												'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
												'':scope.row.have_connected==2,
												'conn_exc':scope.row.connection_status=='Exception',
												'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
										</template>
									</el-table-column>
									<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number" sortable></el-table-column>
									<el-table-column label='<%=rb.getString("HostName")%>' prop="host_name" sortable></el-table-column>
									<el-table-column label='<%=rb.getString("ChanPinLeiXing")%>' prop="product_type" sortable></el-table-column>
									<el-table-column label='<%=rb.getString("ZuiXinGengXinWenJian")%>' prop="latest_update_file"></el-table-column>
									<el-table-column label='<%=rb.getString("ZuiXianGengXinShiJian")%>' prop="latest_update_time" sortable></el-table-column>
									<!-- 模糊查询 -->
									<template slot="toolbar">
					               		<div class="gnbBackupSearchCon commonFlex">
											<el-input placeholder="<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>" suffic-icon='el-icon-search' v-model="gnbAllDevicesSearch">
												<i slot="suffix" class="el-icon el-icon-common-search"  @click="gnbDeviceQueryAll"></i>
											</el-input>
											<div class="tableHeadQueryBoxCls" style='padding: 0 20px;'>
                                                <div v-for="(item,index) in advancedQueryItemList" style='margin: 0;'>
                                                    <div v-if="item.type == 'checkbox' && item.isShow" style="margin-right:10px;">
														<el-popfilter
															:label='item.label'
															v-model="item.checkedItemList"
															:list="item.options"
															:visible.sync="item.isShow"
															@check-change="advanceQuery(item.type,item.value,item.checkedItemList)">
														</el-popfilter>
													</div>
													<div v-if="item.type == 'select' && item.isShow" style="margin-right:10px;">
														<el-popfilter
															type="single"
															:label='item.label'
															v-model="item.selectVal"
															:list="item.options"
															:visible.sync="item.isShow"
															@check-change="advanceQuery(item.type,item.value,item.selectVal)">
														</el-popfilter>
													</div>
                                                </div>
                                                <div class="advancedQueryItemBox"  style="background: #FFF;" @click="clearFilterClick">
                                                    <%=rb.getString("QingKongShaiXuan")%>
                                                </div>
                                            </div>
										</div>
					                </template>
								</el-ctable>
							</div>
						</div>
					</div>
				</div>
				<el-form-item prop='cellCodes' class="noSelectDevice">
					<el-input v-model='gnbRuleForm.cellCodes' v-show="false"></el-input>
				</el-form-item>
			</div>

			<!-- 执行模式 -->
			<div class="selectDeviceWarp">
				<div class="group-title not-extend executeModeInfor">
					<span class="title-icon"></span>
					<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
				</div>
				<div v-if="gnbAddTaskType == 'backup' || gnbAddTaskType == 'restore'"  class="executeModeBox" >
					<el-form-item prop='executeMode'>
						<el-radio-group v-model="gnbRuleForm.executeMode">
							<el-radio label="active"><%=rb.getString("LiJiZhiXing")%></el-radio>
							<el-radio label="timing" style='margin-bottom:0px;margin-left:80px;'><%=rb.getString("DingShiZhiXing")%></el-radio>
						</el-radio-group>
					</el-form-item>
					<el-form-item prop='startTime' class='timeSelect'>
						<el-date-picker
							value-format="yyyy-MM-dd HH:mm:ss"
							v-model='gnbRuleForm.startTime'
							:disabled="gnbSetTimeEnable"
							type="datetime"
							@focus='gnbSetTime'
							:picker-options="pickerOptions">
						</el-date-picker>
					</el-form-item>
				</div>
				<!--周期备份任务-->
				<div v-else class="ftpServerContainer" style="gap:0 180px">
					<!-- Execute 下拉框 -->
					<el-form-item label='<%=rb.getString("ZhouQiZhiXing")%>' prop='executeMode' label-position="top" class="commonWidthItem">
						<el-select v-model='gnbRuleForm.executeMode'>
							<el-option label='<%=rb.getString("Tian")%>' value="every day"></el-option>
							<el-option label='<%=rb.getString("Zhou")%>' value="every week"></el-option>
							<el-option label='<%=rb.getString("Yue")%>' value="every month"></el-option>
						</el-select>
					</el-form-item>
										
					<el-form-item v-if="gnbRuleForm.executeMode == 'every week'" label='<%=rb.getString("Zhou")%>' prop='periodTimeWeek' class="commonWidthItem" label-position="top">
						<el-popover ref="weekPopover"
							placement="bottom-start"
							width="380"
							trigger="click"
							popper-class="week-day-popover">
							<div style="padding: 16px 20px;">
								<div style="display:grid;grid-template-columns:repeat(7,1fr);gap:10px 15px;">
									<div v-for="week in weekDays" :key="week.value" class="selectDateItem" @click="toggleWeekDay(week.value)"
										:style="{
											width: '32px',
											height: '32px',
											lineHeight: '32px',	
											backgroundColor: gnbRuleForm.weekDay.includes(week.value) ? '#4D84FF' : '#FFFFFF',
											color: gnbRuleForm.weekDay.includes(week.value) ? '#FFF' : '#606266',
										}"
										@mouseover="$event.currentTarget.style.transform='translateY(-2px)'"
										@mouseout="$event.currentTarget.style.transform='translateY(0)'">
										{{week.label}}
									</div>
								</div>
							</div>
							<el-input slot="reference" 
								:value="getWeekDayLabels()"
								placeholder=""
								prefix-icon="el-icon-date"
								readonly
								style="cursor:pointer;">
							</el-input>
						</el-popover>
					</el-form-item>

					<!-- Month 月份选择器 -->
					<el-form-item v-if="gnbRuleForm.executeMode == 'every month'" label='<%=rb.getString("Yue")%>' prop='periodTimeMonth' class="commonWidthItem" label-position="top">
						<el-popover ref="monthSelectPopover"
							placement="bottom-start"
							width="300"
							trigger="click"
							popper-class="month-select-popover">
							<div style="padding: 20px;">
								<div style="display:grid;grid-template-columns:repeat(6,1fr);gap:10px 15px;">
									<div v-for="month in 12" :key="month" class="selectDateItem"
										@click="toggleMonth(month)"
										:style="{
											width: '30px',
											height: '30px',
											lineHeight: '30px',
											backgroundColor: gnbRuleForm.months.includes(month + '') ? '#4D84FF' : '#FFFFFF',
											color: gnbRuleForm.months.includes(month + '') ? '#FFFFFF' : '#606266',
										}"
										@mouseover="$event.currentTarget.style.transform='scale(1.05)'"
										@mouseout="$event.currentTarget.style.transform='scale(1)'">
										{{month}}
									</div>
								</div>
							</div>
							<el-input slot="reference" 
								:value="gnbRuleForm.months.length > 0 ? gnbRuleForm.months.map(m => m).join(', ') : ''"
								placeholder=""
								prefix-icon="el-icon-date"
								readonly
								style="cursor:pointer;">
							</el-input>
						</el-popover>
					</el-form-item>

					<!-- Month 选择器 -->
					<el-form-item v-if="gnbRuleForm.executeMode == 'every month'" label='<%=rb.getString("Tian")%>' prop='periodTimeMonthDay' class="commonWidthItem" label-position="top">
						<el-popover ref="monthDayPopover"
							placement="bottom-start"
							width="398"
							trigger="click"
							popper-class="month-day-popover">
							<div style="padding: 20px;">
								<div style="display:grid;grid-template-columns:repeat(8,1fr);gap:10px 15px;">
									<div v-for="day in 31" :key="day" class="selectDateItem"
										@click="toggleMonthDay(day)"
										:style="{
											width: '30px',
											height: '30px',
											lineHeight: '30px',
											backgroundColor: gnbRuleForm.monthDay.includes(day + '') ? '#4D84FF' : '#FFFFFF',
											color: gnbRuleForm.monthDay.includes(day + '') ? '#FFFFFF' : '#606266',
										}"
										@mouseover="$event.currentTarget.style.transform='scale(1.1)'"
										@mouseout="$event.currentTarget.style.transform='scale(1)'">
										{{day}}
									</div>
								</div>
							</div>
							<el-input slot="reference" 
								:value="gnbRuleForm.monthDay.length > 0 ? gnbRuleForm.monthDay.join(', ') : ''"
								placeholder=""
								prefix-icon="el-icon-date"
								readonly
								style="cursor:pointer;">
							</el-input>
						</el-popover>
					</el-form-item>

					<!-- Time 时间选择器 -->
					<el-form-item label='<%=rb.getString("ShiJian")%>' prop='periodTime' class="commonWidthItem" label-position="top">
						<el-time-picker 
							ref="periodTimePicker"
							value-format="HH:mm:ss" 
							v-model='gnbRuleForm.periodTime' 
							@focus="gnbDefaultStartTimes"
							:picker-options="pickerOptions">
						</el-time-picker>
					</el-form-item>
				</div>
			</div>
			<!-- ftp server: periodBackup,updatePeriodBackup 新建后修改周期备份任务才会显示 -->
			<div v-if='gnbAddTaskType == "periodBackup" || gnbAddTaskType == "updatePeriodBackup"' class="executeModeWarp">
				<div class="group-title not-extend executeModeInfor">
					<span class="title-icon"></span>
					<span class="title-text">FTP Server</span>
				</div>
				<el-form-item label='<%=rb.getString("SheZhiKaiGuan")%>' prop='ftpSwitch' label-width='40px' class="executeModeBox">
					<el-switch v-model="gnbRuleForm.ftpSwitch" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
				</el-form-item>
				<div class="ftpServerContainer" style="padding: 0 0 0 76px;">
					<el-form-item label='<%=rb.getString("FTPXieYi")%>' prop="ftpProtocol" label-position="top" class="commonWidthItem">
						<el-select v-model="gnbRuleForm.ftpProtocol" class="language">
							<el-option label='SFTP' value="sftp"></el-option>
							<el-option label='FTP' value="ftp"></el-option>
						</el-select>
					</el-form-item>
					<el-form-item label='<%=rb.getString("ShangChuanLuJing")%>' prop="ftpPath" label-position="top">
						<el-input v-model="gnbRuleForm.ftpPath"></el-input>
					</el-form-item>
					
					<el-form-item label='<%=rb.getString("IPDiZhi")%>' prop="ftpIp" label-position="top">
						<el-input v-model="gnbRuleForm.ftpIp"></el-input>
					</el-form-item>
					<el-form-item label='<%=rb.getString("DuanKou")%>' prop="ftpPort" label-position="top">
						<el-input v-model="gnbRuleForm.ftpPort"></el-input>
					</el-form-item>
					
					<el-form-item label='<%=rb.getString("YongHuMingCheng")%>' prop="ftpUser" label-position="top">
						<el-input v-model="gnbRuleForm.ftpUser" maxlength="60"></el-input>
					</el-form-item>
					<el-form-item label='<%=rb.getString("MiMa")%>' prop="ftpPassword" label-position="top">
						<el-password v-model="gnbRuleForm.ftpPassword" size="mini" show-password placeholder=""></el-password>
						<el-input v-model="gnbRuleForm.ftpPassword" style="display: none;"></el-input>
					</el-form-item>
				</div>
			</div>
		</el-form>
	</div>
	<!--batch input-->
	<el-dialog class='dialogStyle' title='<%=rb.getString("TianJia")%>' width='630px' :visible.sync='gnbBatchSnDialog' :append-to-body="true" :close-on-click-modal="false" @close='gnbCloseBatchSn'>
		<el-form ref='gnbBatchSnForm' :rules='gnbBatchSnRules' :model='gnbBatchSnForm' label-position="top">
			<div>
				<label><%=rb.getString("Title_SheBeiBianMa")%></label>
				<el-form-item prop='serialNumber' style="margin-bottom:22px;">
					<el-input v-model='gnbBatchSnForm.serialNumber' type='textarea' :rows="4" style='margin-top:5px;'></el-input>
				</el-form-item>
				<span class="commonNotes12"><%=rb.getString("eNBZhuCeTiShiWenZi") %></span>
			</div>
			<div style='margin-top:45px;'>
				<el-button @click='gnbSaveBatchSn' type="primary"><%=rb.getString("QueDing")%></el-button>
				<el-button @click='gnbCloseBatchSn'><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-form>
	</el-dialog>
</div>

<script type="text/javascript">
	
var gnbAddViewBackupRestoreVue = new Vue({
	el:'#gnbBackupRestoreTask',
	data(){
		var vm = this,
			gnbValidateName = (rule,value,callback) => {
				if(vm.gnbAddTaskType == 'backup' || vm.gnbAddTaskType == 'restore'){
					if(value === ''){
						callback(new Error('<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'))
					}else{
						callback();
					}
				}else{
					callback();
				}
			},
			gnbValidateStartTime = (rule,value,callback) => {
				if(vm.gnbAddTaskType == 'backup' || vm.gnbAddTaskType == 'restore'){
					if(vm.gnbRuleForm.executeMode !== 'timing'){
						callback()
					}else{
						if(value === '' || value === null){
							callback(new Error('<%=rb.getString("QingXuanZeShiJian")%>'))
						}else{
							callback();
						}
					}
				}else{
					callback()
				}
			},

			gnbValidatePeriodTime = (rule,value,callback) => {
				// 只有周期备份任务且开关打开时才校验必填
				// 普通备份和恢复任务不校验
				if(vm.gnbAddTaskType == 'backup' || vm.gnbAddTaskType == 'restore'){
					callback();
				}else {
					if(vm.gnbRuleForm.enable == '1'){
						if(value === '' || value === null){
							callback(new Error('<%=rb.getString("QingXuanZeShiJian")%>'))
						}else{
							callback();
						}
					}else{
						callback();
					}
				}
			},
			gnbValidateCodes = (rule,value,callback) => {
				// 如果选择"gnbDeviceType 1-全部设备"，不需要校验
				if(vm.gnbDeviceType == '1'){
					callback();
				}else{
					// 普通备份/恢复任务：直接校验必填
					if(vm.gnbAddTaskType == 'backup' || vm.gnbAddTaskType == 'restore'){
						if(value == '' || value == null){
							callback(new Error('<%=rb.getString("QingXuanZeSheBei")%>'));
						}else{
							callback();
						}
					}else{
						// 周期备份任务：只有当开关打开时才校验必填
						if(vm.gnbRuleForm.enable == '1'){
							if(value == '' || value == null){
								callback(new Error('<%=rb.getString("QingXuanZeSheBei")%>'));
							}else{
								callback();
							}
						}else{
							callback();
						}
					}
				}
			},
			
			// FTP字段条件校验：仅当ftpSwitch开启时必填
			validateFtpPath = (rule,value,callback) => {
				if(vm.gnbRuleForm.enable == '1' && vm.gnbRuleForm.ftpSwitch == '1'){
					if(!value || value == ''){
						callback(new Error('<%=rb.getString("QingShuRuShangChuanLuJing")%>'))
					}else{
						callback();
					}
				}else{
					callback();
				}
			},
			validateFtpIp = (rule,value,callback) => {
				var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
				
				// 开关打开时，校验必填
				if(vm.gnbRuleForm.enable == '1' && vm.gnbRuleForm.ftpSwitch == '1'){
					if(!value || value == ''){
						callback(new Error('<%=rb.getString("XinIPShuRuTiShi")%>'))
						return;
					}
				}
				
				// 无论开关状态，如果有值则校验IP格式
				if(value && value != ''){
					if(!reg.test(value)){
						callback(new Error('<%=rb.getString("XinIPGeShiCuoWu")%>'))
						return;
					}
				}
				
				callback();
			},
			validateFtpPort = (rule,value,callback) => {
				var regNum = /^\d+$/;
				
				// 开关打开时，校验必填
				if(vm.gnbRuleForm.enable == '1' && vm.gnbRuleForm.ftpSwitch == '1'){
					if(!value || value == ''){
						callback(new Error('<%=rb.getString("QingShuRuDuanKou")%>'))
						return;
					}
				}
				
				// 无论开关状态，如果有值则校验端口格式和范围（0-65535）
				if(value && value != ''){
					var portNum = parseInt(value);
					if(!regNum.test(value) || (value < 0 || value > 65535)){ 
						callback(new Error('<%=rb.getString("ChangDuChaoChuFanWei")%>'))
						return;
					}
				}
				
				callback();
			},
			validateFtpUser = (rule,value,callback) => {
				if(vm.gnbRuleForm.enable == '1' && vm.gnbRuleForm.ftpSwitch == '1'){
					if(!value || value == ''){
						callback(new Error('<%=rb.getString("QingShuRuYongHuMing")%>'))
					}else{
						callback();
					}
				}else{
					callback();
				}
			},
			validateFtpPassword = (rule,value,callback) => {
				if(vm.gnbRuleForm.enable == '1' && vm.gnbRuleForm.ftpSwitch == '1'){
					if(!value || value == ''){
						callback(new Error('<%=rb.getString("QingShuRuMiMa")%>'))
					}else{
						callback();
					}
				}else{
					callback();
				}
			},
			// 周选择器校验：executeMode为week时必选
			validateWeekDay = (rule,value,callback) => {
				if(vm.gnbRuleForm.enable == '1' && vm.gnbRuleForm.executeMode == 'every week'){
					if(!vm.gnbRuleForm.weekDay || vm.gnbRuleForm.weekDay.length == 0){
						callback(new Error('<%=rb.getString("QingZhiShaoXuanZeYiTian")%>'))
					}else{
						callback();
					}
				}else{
					callback();
				}
			},
			// 月选择器校验：executeMode为month时必选
			validateMonthDay = (rule,value,callback) => {
				if(vm.gnbRuleForm.enable == '1' && vm.gnbRuleForm.executeMode == 'every month'){
					if(!vm.gnbRuleForm.monthDay || vm.gnbRuleForm.monthDay.length == 0){
						callback(new Error('<%=rb.getString("QingZhiShaoXuanZeYiTian")%>'))
					}else{
						callback();
					}
				}else{
					callback();
				}
			},
			// 月份选择器校验：executeMode为month时必选
			validateMonth = (rule,value,callback) => {
				if(vm.gnbRuleForm.enable == '1' && vm.gnbRuleForm.executeMode == 'every month'){
					if(!vm.gnbRuleForm.months || vm.gnbRuleForm.months.length == 0){
						callback(new Error('<%=rb.getString("QingZhiShaoXuanZeYiTian")%>'))
					}else{
						callback();
					}
				}else{
					callback();
				}
			},
			validatorSn = (rule,value,callback) => {
				var serialNumber = value, temp = /^(\d|[a-zA-Z]|-|\s){1,30}$/,
					list = serialNumber.replace(/[\(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ 
						return item.length > 0;
					});
				if (serialNumber == null || serialNumber.length == 0) {
					callback(new Error('<%=rb.getString("SNBuNengWeiKong")%>'));
				} else {
					var nameFlag = list.every(function(item,index){
						return temp.test(item)
					})
					if(nameFlag){
						callback()
					}else{
						callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
					}
				}
			};
		return {
			gnbLeftUrl:'',
			gnbRightUrl:'',
			gnbDeviceTitle:['<%=rb.getString("BackupRestoreSheBeiLieBiao")%>','<%=rb.getString("YiXuanZeJiZhan")%>'],
			gnbHeight:'300px',
			gnbSetTimeEnable:true,
			gnbRestoreType:'restore',
			gnbRuleForm:{
				taskName:'${addTaskName}',
				taskType:"",
				executeMode:'active',
				startTime:'',
				timeZone:timeZone,
				cellCodes:'',

				enable: '0',
				periodTime:'',
				periodTimeWeek: '',
				weekDay: [],
				periodTimeMonth: '',
				periodTimeMonthDay: '',
				monthDay: [],
				months: [],

				ftpSwitch: '0',
				ftpUser: '',
				ftpIp: '',
				ftpPort: '',
				ftpPath: '',
				ftpPassword: '',
				ftpProtocol: 'sftp',
				cronPeriod: '' //传 cron表达式
			},
			gnbParamsAll:{
				timeZone: timeZone,
				product_type:'',
				searchText:'',
				likeFields: 'serial_number,host_name',
				isGnb: 1
			},
			gnbDeviceSelectParams:{
				timeZone: timeZone,
				product_type:'',
				searchText:'',
				likeFields: 'serial_number,host_name',
				isGnb: 1
			},
			gnbFilterParams: {
				//likeFields: 'serial_number,host_name',
				isUpdatePW: "true",
				isGnb: 1
			},
			product_type:'',
			pickerOptions:{
				disabledDate(time){
					return time.getTime()< Date.now()-8.64e7;
				}
			},
			gnbDeviceUrl:'',
			gnbDeviceType:'2',
			gnbAddTaskType:'',
			weekDays: [
				{label: 'Sun', value: '1'},
				{label: 'Mon', value: '2'},
				{label: 'Tue', value: '3'},
				{label: 'Wed', value: '4'},
				{label: 'Thu', value: '5'},
				{label: 'Fri', value: '6'},
				{label: 'Sat', value: '7'}
			],
			gnbSelection:[],
			gnbUpdateTaskId:'',
			gnbRules:{
				taskName:[
					{validator: gnbValidateName,trigger:'blur'}
				],
				startTime:[
					{type:'date',validator: gnbValidateStartTime,trigger:'change'}
				],
				cellCodes:[
					{validator: gnbValidateCodes,trigger:'change'}
				],
				executeMode: [
					{ required: true, message: '<%=rb.getString("QingXuanZeZhiXingFangShi")%>', trigger: 'change' }
				],
				periodTime:[
					{required: true, type:'date',validator: gnbValidatePeriodTime,trigger:'change'}
				],
				periodTimeWeek: [
					{ required: true, validator: validateWeekDay, trigger: 'change' }
				],
				periodTimeMonth: [
					{ required: true, validator: validateMonth, trigger: 'change' }
				],
				periodTimeMonthDay: [
					{ required: true, validator: validateMonthDay, trigger: 'change' }
				],
				
				ftpProtocol: [
					{ required: true, message: '<%=rb.getString("QingXuanZeFTPXieYi")%>', trigger: 'change' }
				],
				ftpPath: [
					{ required: true, validator: validateFtpPath }
				],
				ftpIp: [
					{ required: true, validator: validateFtpIp }
				],
				ftpPort: [
					{ required: true, validator: validateFtpPort }
				],
				ftpUser: [
					{ required: true, validator: validateFtpUser }
				],
				ftpPassword: [
					{ required: true, validator: validateFtpPassword }
				],
			},
			gnbAllDevicesSearch:'',
			gnbSpecifiedDevicesSearch:'',
			gnbDeviceTypeSpecified:true,
			gnbDeviceTypeAll:false,

			advancedQueryItemList:[
                {
                    type:'checkbox',
                    isShow:true,
                    isIndeterminate:false,
                    checkAll:false,
                    popoverShow:false,
                    checkedItemList:[],
                    oldCheckedItemList:[],
                    label:'<%=rb.getString("ChanPinLeiXing") %>',
                    options:[],
                    value:'product_type',
                },
            ],
			//batch add
			gnbBatchSnDialog: false,
			gnbBatchSnForm:{
				serialNumber: '',
				type: 'input'
			},
			gnbBatchSnRules:{
				serialNumber:[
					{validator: validatorSn, trigger:'change'}
				]
			},
		}
	},
	watch:{
		gnbRestoreType:function(val){
			var vm = this;

			if(vm.gnbAddTaskType == 'restore'){
				vm.gnbDeviceType = '2';
				if(val == 'reset'){
					vm.gnbSpecifiedDevicesSearch = '';
					vm.gnbDeviceSelectParams.searchText = '';
					vm.gnbDeviceSelectParams.isFilterAuxiliary = '1';
					//选中all devices perform 时
					vm.gnbAllDevicesSearch = '';
					vm.gnbParamsAll.searchText = '';
					vm.gnbParamsAll.isFilterAuxiliary = '1';
				}else{
					//全部设备
					vm.gnbAllDevicesSearch = '';
					vm.gnbParamsAll.searchText = '';
					delete vm.gnbParamsAll.isFilterAuxiliary;
					//已选设备
					vm.gnbSpecifiedDevicesSearch = '';
					vm.gnbDeviceSelectParams.searchText = '';
					delete vm.gnbDeviceSelectParams.isFilterAuxiliary;
				}
				vm.$nextTick(function(){
					if(vm.$refs.gnbBackupRestoreTaskTable){
						vm.$refs.gnbBackupRestoreTaskTable.reload();
					}
				});
			}
		},
		gnbDeviceType:function(val){
			var vm = this;

			if(val == '1'){
				vm.gnbDeviceTypeAll = true;
				vm.gnbDeviceTypeSpecified = false;
				vm.gnbSpecifiedDevicesSearch = '';
				vm.gnbDeviceSelectParams.searchText = '';
				vm.gnbDeviceSelectParams.product_type = '';
				
				if(vm.$refs.gnbRuleForm){
					vm.$refs.gnbRuleForm.clearValidate('cellCodes');
				}
			}else{
				vm.gnbDeviceTypeSpecified = true;
				vm.gnbDeviceTypeAll = false;
				vm.gnbAllDevicesSearch = '';
				vm.gnbParamsAll.searchText = '';
				vm.gnbParamsAll.product_type = '';
			}
			vm.clearFilterClick();
		},
		gnbSelection(){
			var vm = this,
				data = vm.$refs.gnbBackupRestoreTaskTable.getData();

			if(data.length != 0){
				vm.gnbRuleForm.cellCodes = data.map(function(item){
					return item.small_cell_code;
				}).join(',');
			}else{
				vm.gnbRuleForm.cellCodes = '';
			}
		},

		"gnbRuleForm.executeMode":function(newVal){
			var vm = this;

			if(vm.gnbAddTaskType == 'backup' || vm.gnbAddTaskType == 'restore'){
				// 根据 executeMode 的值设置时间选择器的启用/禁用状态
				if(newVal == 'timing'){
					vm.gnbSetTimeEnable = false
				}else{
					vm.gnbSetTimeEnable = true
				}
				vm.$refs.gnbRuleForm.validateField('startTime')
			}else{
				vm.$refs.weekPopover && vm.$refs.weekPopover.doClose();
				vm.$refs.monthSelectPopover && vm.$refs.monthSelectPopover.doClose();
				vm.$refs.monthDayPopover && vm.$refs.monthDayPopover.doClose();
				vm.$refs.periodTimePicker && vm.$refs.periodTimePicker.hidePicker();

				vm.gnbRuleForm.weekDay = [];
				vm.gnbRuleForm.monthDay = [];
				vm.gnbRuleForm.months = [];
				vm.gnbRuleForm.periodTime = '';
				
				// 清除验证状态
				if(vm.$refs.gnbRuleForm){
					vm.$nextTick(function(){
						vm.$refs.gnbRuleForm.clearValidate(['periodTime', 'periodTimeWeek', 'periodTimeMonth', 'periodTimeMonthDay']);
					});
				}
			}
		},
		"gnbRuleForm.ftpSwitch":function(newVal){
			// 当FTP开关状态改变时，清除相关字段的验证状态
			var vm = this; 
			if(vm.$refs.gnbRuleForm){
				vm.$nextTick(function(){
					// 清除FTP相关字段的验证提示
					vm.$refs.gnbRuleForm.clearValidate(['ftpProtocol','ftpPath','ftpIp','ftpPort','ftpUser','ftpPassword']);
				});
			}
		},
	},
	methods:{
		gnbInit(){
			var vm = this;

			vm.$refs.gnbBackupRestoreTaskTable.appendCheckedRows(gnbBackupRestoreVue.gnbDeviceSnSelection);
			vm.$nextTick(function(){
				vm.gnbLeftUrl = '${ctx}/task/enb/config/backupRestore/queryCellInfos.action';
				vm.gnbDeviceUrl = '${ctx}/task/enb/config/backupRestore/queryCellInfos.action';
			});

			var params={
                isUpdatePW:"true",
                isGnb: 1
            };

            axios.post( '${ctx}/cell/version/getProductType.action', stringify(params)).then(function(response){
                var data = response.data;

                var arr = [];
                (data || []).map(function(item){
                    if (item){
                        arr.push({label:item.name,value:item.value})
                    }
                })
                vm.advancedQueryItemList.map((items)=>{
                    if('product_type' == items.value){
                        items.options = arr;
                    }
                })
            });
		},
		gnbInitAddTask(type, data){
			var vm = this;

			vm.gnbAddTaskType = type;

			vm.gnbUpdateTaskId = data.task_id;
			
			if(type == 'updatePeriodBackup'){
				if(data.selected_type == 'all'){
					vm.gnbDeviceType = '1';
				}else{
					//指定设备执行
					vm.gnbDeviceType = '2';
					vm.gnbRightUrl = '${ctx}/task/enb/config/backupRestore/periodTask/deviceList.action?taskId=' + vm.gnbUpdateTaskId;
				}

				//周期备份任务的开关
				vm.gnbRuleForm.enable = data.is_enable == 1 ? '1' : '0'; 
				// 回显 FTP Server 数据
				vm.gnbRuleForm.ftpSwitch = data.ftpSwitch || '0';
				vm.gnbRuleForm.ftpUser = data.ftpUser || '';
				vm.gnbRuleForm.ftpIp = data.ftpIp || '';
				vm.gnbRuleForm.ftpPort = data.ftpPort || '';
				vm.gnbRuleForm.ftpPath = data.ftpPath || '';
				vm.gnbRuleForm.ftpPassword = data.ftpPassword || '';
				vm.gnbRuleForm.ftpProtocol = data.ftpProtocol || 'sftp';
				
				// 回显执行模式相关数据
				vm.gnbRuleForm.executeMode = data.executeMode || 'every day';
			}
			vm.$nextTick(function(){
				// 根据任务类型设置 executeMode 默认值（在 nextTick 中设置，确保表单已初始化）
				if(type == 'periodBackup'){
					// 新建周期备份任务，默认为 every day
					vm.gnbRuleForm.executeMode = 'every day';
				}else if(type == 'updatePeriodBackup'){
					// 新建周期备份任务，默认为 every day
					vm.gnbRuleForm.executeMode = data.executeMode || 'every day';

					// periodTime 始终从 cronPeriod 中解析（在 nextTick 中执行，确保 DOM 已更新）
					if(data.cronPeriod){
						vm.gnbParseCronExpression(data.cronPeriod, data.executeMode);
					}
				}else{
					// 普通备份/恢复任务，默认为 active
					vm.gnbRuleForm.executeMode = 'active';
				}
				
				initForm(vm.$refs.gnbRuleForm);
	    	});
		},	

		// 天，周，月： 秒 分 时 日 月 周
		gnbGetCronExpression(){
			var vm = this,
				time = vm.gnbRuleForm.periodTime || '00:00:00',
				timeParts = time.split(':'),
				second = timeParts[2] || '0',
				minute = timeParts[1] || '0',
				hour = timeParts[0] || '0',
				cronPeriod = '';
			
			switch(vm.gnbRuleForm.executeMode){
				case 'every day':
					// day: 秒 分 时 日 月 周 
					cronPeriod = second + ' ' + minute + ' ' + hour + ' * * ?';
					break;
					
				case 'every week':
					// week: 秒 分 时 日 月 周
					// 例如: 10 10 10 ? * 1,3,5 (每周一、三、五10:10:10执行)
					var weekDays = vm.gnbRuleForm.weekDay.length > 0 ? vm.gnbRuleForm.weekDay.join(',') : '*';
					cronPeriod = second + ' ' + minute + ' ' + hour + ' ? * ' + weekDays;
					break;  
					
				case 'every month':
					// month: 秒 分 时 日 月 周 
					// 月份在第5位，日期在第4位
					var monthDays = vm.gnbRuleForm.monthDay.length > 0 ? vm.gnbRuleForm.monthDay.join(',') : '*';
					var months = vm.gnbRuleForm.months.length > 0 ? vm.gnbRuleForm.months.join(',') : '*';
					cronPeriod = second + ' ' + minute + ' ' + hour + ' ' + monthDays + ' ' + months + ' ?';
					break;
					
				default:
					cronPeriod = second + ' ' + minute + ' ' + hour + ' * * ?';
			}
			
			return cronPeriod;
		},
		// 解析 cron 表达式并回显到页面
		gnbParseCronExpression(cronPeriod, executeMode){
			var vm = this;
			
			// 如果没有 cronPeriod，直接返回
			if(!cronPeriod){
				return;
			}
			
			// 解析 cron 表达式: 秒 分 时 日 月 周 年
			var cronParts = cronPeriod.trim().split(/\s+/);
			
			if(cronParts.length < 6){
				return;
			}
			
			var second = cronParts[0] || '0',
				minute = cronParts[1] || '0',
				hour = cronParts[2] || '0',
				dayOfMonth = cronParts[3] || '*',
				month = cronParts[4] || '*',
				dayOfWeek = cronParts[5] || '?';
			
			// 格式化时间为两位数（例如：09:05:03）
			var formatTime = function(val){
				return String(val).length === 1 ? '0' + val : String(val);
			};
			
			// 回显时间（时:分:秒）- 始终从 cron 表达式中解析
			var timeStr = formatTime(hour) + ':' + formatTime(minute) + ':' + formatTime(second);
						
			// 使用 Vue.set 确保响应式更新
			vm.$set(vm.gnbRuleForm, 'periodTime', timeStr);
			

			// 根据 executeMode 回显对应的日期选择
			if(executeMode === 'every week'){
				// 解析周几（dayOfWeek）
				if(dayOfWeek && dayOfWeek !== '?' && dayOfWeek !== '*'){
					vm.gnbRuleForm.weekDay = dayOfWeek.split(',');
				}
			}else if(executeMode === 'every month'){
				// 解析每月的日期（dayOfMonth）和月份（month）
				if(dayOfMonth && dayOfMonth !== '?' && dayOfMonth !== '*'){
					vm.gnbRuleForm.monthDay = dayOfMonth.split(',');
				}
				if(month && month !== '?' && month !== '*'){
					vm.gnbRuleForm.months = month.split(',');
				}
			}
		},
		toggleWeekDay(dayValue){
			var vm = this,
				index = vm.gnbRuleForm.weekDay.indexOf(dayValue);

			if(index > -1){
				vm.gnbRuleForm.weekDay.splice(index, 1);
			}else{
				vm.gnbRuleForm.weekDay.push(dayValue);
			}
			
			// 触发验证
			vm.$nextTick(function(){
				if(vm.$refs.gnbRuleForm){
					vm.$refs.gnbRuleForm.validateField('periodTimeWeek');
				}
			});
		},
		getWeekDayLabels(){
			var vm = this;

			if(vm.gnbRuleForm.weekDay.length === 0) return '';

			var labels = vm.gnbRuleForm.weekDay.map(function(val){
				var day = vm.weekDays.find(function(d){ return d.value === val; });
				return day ? day.label : '';
			});
			return labels.join(', ');
		},
		toggleMonth(month){
			var vm = this;
				monthStr = month + '',
				index = vm.gnbRuleForm.months.indexOf(monthStr);

			if(index > -1){
				vm.gnbRuleForm.months.splice(index, 1);
			}else{
				vm.gnbRuleForm.months.push(monthStr);
			}
			
			// 触发验证
			vm.$nextTick(function(){
				if(vm.$refs.gnbRuleForm){
					vm.$refs.gnbRuleForm.validateField('periodTimeMonth');
				}
			});
		},
		toggleMonthDay(day){
			var vm = this;
				dayStr = day + '',
				index = vm.gnbRuleForm.monthDay.indexOf(dayStr);

			if(index > -1){
				vm.gnbRuleForm.monthDay.splice(index, 1);
			}else{
				vm.gnbRuleForm.monthDay.push(dayStr);
			}
			
			// 触发验证
			vm.$nextTick(function(){
				if(vm.$refs.gnbRuleForm){
					vm.$refs.gnbRuleForm.validateField('periodTimeMonthDay');
				}
			});
		},
		gnbDeviceSelectQuery(val){
			var vm = this;

			vm.gnbDeviceSelectParams.searchText = vm.gnbSpecifiedDevicesSearch;
		},
		gnbDeviceQueryAll(val){
			var vm = this;

			vm.gnbParamsAll.searchText = vm.gnbAllDevicesSearch;
		},
		/**
		 * 全选操作
		 * @param gnbSelection:选择的数据
		*/
		gnbSelectChange(selection){
			var vm = this;

			vm.gnbSelection = selection;
		},

		//batch add 
		gnbBatchBtnClick(){
			var vm = this;

			vm.gnbBatchSnDialog = true;
			//清空表单
			if(vm.$refs.gnbBatchSnForm){
				vm.$refs.gnbBatchSnForm.resetFields();
			}
		},
		gnbSaveBatchSn(){
			var vm = this,  
				params = {}, 
				snStr = vm.gnbBatchSnForm.serialNumber || '',
				list = snStr.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;});
			
			params.selectedCells = list.join(",");
			params.isGnb = 1;

			vm.$refs.gnbBatchSnForm.validate((valid) => {
				if(valid){
					axios.post('${ctx}/task/enb/config/backupRestore/queryCellInfos.action', stringify(params)).then((res)=>{
						var data = res.data;
						
						if(data && data.length > 0){		
							vm.$refs.gnbBackupRestoreTaskTable.appendCheckedRows(data);
							vm.gnbCloseBatchSn();
						}else{
							vm.$message('<%=rb.getString("MeiYouKePiPeiSheBei")%>');
						}							
					})
				}
			})
		},
		gnbCloseBatchSn(){
			var vm = this;

			vm.gnbBatchSnDialog = false;
			vm.$refs.gnbBatchSnForm.resetFields(); 
		},

		//保存
		gnbTaskSubmit(){
	    	var vm = this, 
                message = '<%=rb.getString("ChengGong")%>';
            // 防止多次提交
            if(gnbBackupRestoreVue.slideSubmitLoading)return

	    	vm.$refs.gnbRuleForm.validate((valid) => {
	    		if(valid){
	    			var params = {},
						url;
	    			//新建备份或恢复任务
					if(vm.gnbAddTaskType == 'backup' || vm.gnbAddTaskType == 'restore'){
						params.taskName = vm.gnbRuleForm.taskName;

		    			params.executeMode = vm.gnbRuleForm.executeMode;
		    			if(vm.gnbRuleForm.executeMode == 'timing'){
		    				params.startTime = vm.gnbRuleForm.startTime;
		    			}
		    			params.timeZone = timeZone;
		    			//新建恢复任务：新增 gnbRestoreType 参数
		    			if(vm.gnbAddTaskType == 'restore'){
			    			params.restoreType = vm.gnbRestoreType;
			    			//新建恢复任务类型：
			    			if(vm.gnbRestoreType == 'restore'){
			    				params.taskType = 'restore';
			    			}else{
			    				params.taskType = 'reset';
			    			}
		    			}else{
		    				// 新建备份任务：类型
		    				params.taskType = 'backup';
		    			}

		    			if(vm.gnbDeviceType == '1'){
		    				params.selectAll = "true";
		    			}else{
		    				params.selectAll = "false";
		    				params.cellCodes = vm.gnbRuleForm.cellCodes;
		    			}
		    			params.isGnb = 1;
		    			url = '${ctx}/task/enb/config/backupRestore/addBackupRestoreTask.action';

					}else if(vm.gnbAddTaskType == 'periodBackup'){
						//新建周期备份
						params.taskName = '';
						params.periodTime = vm.gnbRuleForm.periodTime;
		    			params.timeZone = timeZone;
						//周期备份开关
						params.enable = vm.gnbRuleForm.enable;

		    			if(vm.gnbDeviceType == '1'){
		    				params.selectAll = "true";
		    			}else{
		    				params.selectAll = "false";
		    				params.cellCodes = vm.gnbRuleForm.cellCodes;
		    			}

						params.executeMode =  vm.gnbRuleForm.executeMode;
						params.cronPeriod = vm.gnbGetCronExpression();

						// FTP Server 和执行模式 - 解构赋值
						params.ftpSwitch = vm.gnbRuleForm.ftpSwitch;
						params.ftpUser = vm.gnbRuleForm.ftpUser;
						params.ftpIp = vm.gnbRuleForm.ftpIp;
						params.ftpPort = vm.gnbRuleForm.ftpPort;
						params.ftpPath = vm.gnbRuleForm.ftpPath;
						params.ftpPassword = vm.gnbRuleForm.ftpPassword;
						params.ftpProtocol = vm.gnbRuleForm.ftpProtocol;	
											
						params.isGnb = 1;
						url = '${ctx}/task/enb/config/backupRestore/addOrUpdatePeriodBackupTask.action';
					}else{
						//修改周期备份
						params.taskName = '';

						params.periodTime = vm.gnbRuleForm.periodTime;
		    			params.timeZone = timeZone;
						//周期备份开关
						params.enable = vm.gnbRuleForm.enable;
		    			params.taskId = vm.gnbUpdateTaskId;

		    			if(vm.gnbDeviceType == '1'){
		    				params.selectAll = "true";
		    			}else{
		    				params.selectAll = "false";

		    				var rows = vm.$refs.gnbBackupRestoreTaskTable.getData();
		    				if(rows.length != 0){
		    					params.cellCodes = rows.map(function(row){ 
									return row.small_cell_code;
								}).join(',');
		    				}
		    			}
						params.executeMode = vm.gnbRuleForm.executeMode;
						params.cronPeriod = vm.gnbGetCronExpression();

						// FTP Server 和执行模式 - 解构赋值
						params.ftpSwitch = vm.gnbRuleForm.ftpSwitch;
						params.ftpUser = vm.gnbRuleForm.ftpUser;
						params.ftpIp = vm.gnbRuleForm.ftpIp;
						params.ftpPort = vm.gnbRuleForm.ftpPort;
						params.ftpPath = vm.gnbRuleForm.ftpPath;
						params.ftpPassword = vm.gnbRuleForm.ftpPassword;
						params.ftpProtocol = vm.gnbRuleForm.ftpProtocol;
						url='${ctx}/task/enb/config/backupRestore/addOrUpdatePeriodBackupTask.action';
					}
                    gnbBackupRestoreVue.slideSubmitLoading = true;
					axios.post(url,stringify(params)).then(function(response){
	    				var data = response.data;
	    				if(data["success"]){
	    					vm.$message({
	    						message:'<%=rb.getString("ChengGong")%>',
	    						type:'success',
	    					})
                            eventBus.$emit('cancel-gnbBackupRestore');
	    				}else{
	    					vm.$message.error(data["message"]);
                            gnbBackupRestoreVue.slideSubmitLoading = false;
	    				}
	    			})
	    		}else{
	    			return false;
	    		}
	    	})
		},


		// 默认开始时间-时分秒
		gnbDefaultStartTimes(){
			var vm = this;

			if(!vm.gnbRuleForm.startTime){
				vm.gnbRuleForm.startTime = formatDate(Date.getNow()).slice(-8)
			}
		},
		// 设置时间-年月日 时分秒
		gnbSetTime(){
			var vm = this;

			vm.gnbRuleForm.startTime = formatDate(new Date(gloableTime));
			vm.$refs.gnbRuleForm.validateField('startTime');
		},
		// 关闭新建弹窗
		gnbCancel(){
			var vm = this, confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>';

			if(isFormChanged(vm.$refs.gnbRuleForm)){
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					eventBus.$emit('cancel-gnbBackupRestore');
				}).catch(() => {

				})
			}else{
				eventBus.$emit('cancel-gnbBackupRestore');
			}
		},

		//产品类型筛选---------------------------------
		// 高级查询 确定事件
		advanceQuery(type,paramsItem,value){
			var vm = this,
				params ={};
			if(type == 'select'){
				params[paramsItem] = value;
			}else{
				params[paramsItem] = value.join(',');
			}
			if(vm.gnbDeviceType == '1'){
				Object.assign(vm.gnbParamsAll, params);
		   }else{
				Object.assign(vm.gnbDeviceSelectParams, params);
		   }
		},
        // 清除筛选
        clearFilterClick(){
            var vm = this,
                params = {};
            vm.advancedQueryItemList.map((items)=>{
                if(items.isShow && items.isShow== true ){
                    if(items.type == 'checkbox'){
                        items.checkedItemList = [];
                        items.oldCheckedItemList = [];
                        items.checkAll = false;
                        items.isIndeterminate = false;
                    }else if(items.type == 'select'){
                        items.selectVal = '';
                    }
                    if(items.type == 'checkbox' || items.type == 'select'){
                        params[items.value] = '';
                    }
                }
            })
            if(vm.gnbDeviceType == '1'){
                 Object.assign(vm.gnbParamsAll, params);
            }else{
                 Object.assign(vm.gnbDeviceSelectParams, params);
            }
            document.body.click();
        },
	},

	mounted(){
		this.gnbInit();
		eventBus.$off('taskSave-gnbOk').$on('taskSave-gnbOk',this.gnbTaskSubmit);
		eventBus.$off("show-gnbType").$on("show-gnbType",this.gnbInitAddTask);
		eventBus.$off('hander-gnbCancel').$on('hander-gnbCancel',this.gnbCancel);
	}
})
</script>
