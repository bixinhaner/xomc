<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<%
	UserInfo user = (UserInfo) session.getAttribute(ComConstants.SESSION_KEY);
%>

<style>
#mrPage .el-tabs__nav-wrap::after {
	right:120px;
}
#mrPage .el-pagination .el-select .el-input{
	width:85px;
}
#mrPage .el-pagination .el-select .el-input .el-input__inner{
	width:85px;
	background:#fff !important;
}

#mrPage .h100{height: 100%}
#mrPage .curpo{
	cursor: pointer
}
#mrPage .el-icon-status-terminate:before,
#mrPage .el-icon-status-waiting1:before,
#mrPage .el-icon-status-inProgress:before{
	color:#4D84FF;
}
#mrPage .el-icon-status-success:before {
	color:#67D972;
}
#mrPage .el-icon-status-failed:before {
	color:#E88282;
}
#mrPage .curStatus{
	font-size:18px;
	margin-right:5px;
}
#mrPage .el-tabs__header{
	border: 1px solid #E4E7EC;
}
#mrPage .el-tabs__item{
	height: 36px;
	line-height: 36px;
}
#mrPage .el-ctable-toolbar{
	padding: 0px!important;
}
#mrPage .commonQuery .el-date-editor .el-range__close-icon{
	line-height: 20px;
}
.mrTaskResultSlideCls .el-card__header .el-icon-close::before{
    font-size: 16px;
    position: relative;
    top: -10px;
}
</style>
<div class="overflow-cls">
    <div class="panelDefault overHide commonWarp" id='mrPage' style="min-width: 900px;position: relative;overflow: hidden;">
        <!-- 按钮  -- 新建任务 -->
        <div @click="addMrTask" class="newIconBoxCls-bt CODE_PERFORMANCE_MR hidden" style="right:10px;top:5px;" v-show="activeName == 'first'" placeholder='<%=rb.getString("TianJia")%>'>
            <span class="el-icon-circle-add el-icon"></span>
        </div>

        <!-- 主页面区域 -- tab页 -->
        <el-tabs v-model="activeName" @tab-click='tabClick' class="h100">
            <!-- MR 任务列表 -->
            <el-tab-pane label='<%=rb.getString("MRRenWuGuanLi")%>' name="first">
                <el-ctable id="mrTaskTableList" ref="mrTaskTableList" :time="6" :url="mrTaskTableUrl" :height="height" 
                    :row-key="'task_id'" :query-params="queryTaskParams" pagination="true" :rownumber='true' >
                    <!-- 高级查询 -- 设备上报日志-->
                    <template slot="toolbar">
                        <div class='commonQuery' style='display: flex;align-items: center;height:45px;'>
                            <el-query type="normal" @query="queryMrTask" placeholder='<%=rb.getString("RenWuMingCheng")%> / <%=rb.getString("ChuangJianZhe")%>'></el-query>  
                            <el-date-picker style='margin-left: 10px;' 
                                v-model="queryTaskTime"
                                type="datetimerange"
                                value-format="yyyy-MM-dd HH:mm:ss"
                                range-separator="—"  
                                @change="queryTaskTimeChange"
                                start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
                                end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
                            </el-date-picker>
                            <el-popfilter style="margin-left: 10px;"
                                type="single"
                                label='<%=rb.getString("ZhuangTai") %>'
                                v-model="queryTaskParams.taskStatus"
                                :list="mrTaskStatusList.map(item=>{return {label:item.text,value:item.value}})">
                            </el-popfilter>
                            <el-popfilter style="margin-left: 10px;"
                                type="single"
                                label='<%=rb.getString("JieGuo") %>'
                                v-model="queryTaskParams.taskResult"
                                :list="mrTaskResultList.map(item=>{return {label:item.text,value:item.value}})">
                            </el-popfilter>  
                            <div class="pop-filter-clear" style="margin: 0 10px;" 
                                @click="resetQuery">
                                <%=rb.getString("QingKongShaiXuan")%>
                            </div>
                        </div>
                    </template>
                    <!-- 主列表 -->
                    <el-table-column label='' width="30" prop="" class-name="no-text-tips">
                        <template slot-scope="scope">
                            <div class="el-icon el-icon-operation-more curpo" @click="mrTaskOptClick(scope.row,event)" v-clickoutside="handerClose"></div>
                        </template>
                    </el-table-column>
                    <el-table-column label='<%=rb.getString("RenWuMingCheng")%>' min-width="150" prop="task_name"></el-table-column>
                    <el-table-column label='<%=rb.getString("ChuangJianZhe")%>' min-width="80" prop="creator"></el-table-column>
                    <el-table-column label='<%=rb.getString("ChuangJianShiJian")%>' min-width="100" prop="create_time"></el-table-column>
                    
                    <el-table-column prop="task_status" label='<%=rb.getString("ZhuangTai")%>' min-width="80">
                        <template slot-scope="scope">
                            <div v-if="scope.row.task_status == 'waitting'">
                                <span class="el-icon el-icon-status-waiting1 curStatus"></span>
                                <span><%=rb.getString("DengDaiZhiXing")%></span>
                            </div>
                            <div v-if="scope.row.task_status == 'on'">
                                <span class="el-icon el-icon-status-inProgress curStatus"></span>
                                <span><%=rb.getString("KaiQi")%></span>
                            </div>
                            <div v-if="scope.row.task_status == 'off'">
                                <span class="el-icon el-icon-status-terminate curStatus"></span>
                                <span><%=rb.getString("GuanBi")%></span>
                            </div>
                            <div v-if="scope.row.task_status == 'suspend'">
                                <span class="el-icon el-icon-status-suspend curStatus"></span>
                                <span><%=rb.getString("ZanTingStatus")%></span>
                            </div>
                            <div v-if="scope.row.task_status == 'termination'">
                                <span class="el-icon el-icon-status-terminate curStatus"></span>
                                <span><%=rb.getString("YiZhongZhi")%></span>
                            </div>
                            <div v-if="scope.row.task_status == 'unspport'">
                                <span>--</span>
                            </div>
                        </template>
                    </el-table-column>				
                    <el-table-column label='<%=rb.getString("JieGuo")%>' min-width="80"  prop="task_result" >
                        <template slot-scope="scope">
                            <div v-if="scope.row.task_result == 'success'">
                                <span><%=rb.getString("ChengGong")%></span>
                            </div>
                            <div v-if="scope.row.task_result == 'partialSuccess'">
                                <span><%=rb.getString("BuFenChengGong")%></span>
                            </div>
                            <div v-if="scope.row.task_result == 'failure'">
                                <span><%=rb.getString("ShiBai")%></span>
                            </div>
                            <div v-if="scope.row.task_result == 'unExecuted'">
                                <span><%=rb.getString("WeiZhiXing")%></span>
                            </div>
                            <div v-if="scope.row.task_result == 'inExecuted'">
                                <span><%=rb.getString("ZhengZaiZhiXing")%></span>
                            </div>
                        </template>
                    </el-table-column>
                    <el-table-column label='<%=rb.getString("KaiShiShiJian")%>' min-width="120" prop="start_time"></el-table-column>
                    <el-table-column label='<%=rb.getString("JieShuShiJian")%>' min-width="120" prop="end_time"></el-table-column>

                </el-ctable>
                <el-cmenu ref="mrTaskMenu" :data="mrTaskMenuList" @click="mrTaskMenuClick"></el-cmenu>
            </el-tab-pane>

            <!-- MR 文件列表 -->
            <el-tab-pane label='<%=rb.getString("MRWenJianGuanLi")%>' name="third">
                <el-ctable id="mrFileTableList" ref="mrFileTableList" :height="height" :url="mrFileTableUrl"
                    :row-key="'smallCellCode'" :time="6" :query-params="queryFileParams" pagination="true" :rownumber='true'>
                    
                    <!-- 查询 -->
                    <template slot="toolbar">
                        <div class='commonQuery' style='display: flex;align-items: center;height:45px;'>
                            <el-query type="normal" @query="queryMrFile" placeholder="<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>"></el-query>  
                        </div>
                    </template>
                    <!-- 主列表 -->
                    <el-table-column label='' width="45">
                        <template slot-scope="scope">
                            <span class="el-icon el-icon-operation-download curpo" @click="downloadMrFile(scope.row,event)" title='<%=rb.getString("XiaZai")%>'></span>
                        </template>
                    </el-table-column>
                    <el-table-column label='<%=rb.getString("ZhuangTai")%>' width="70" prop="status" sortable>
                        <template slot-scope="scope">
                            <div v-if="scope.row.status == '0'">
                                <img src ='${ctx}/css/images/main/monitor-ico/monitor-kpi-offomc.png' title='<%=rb.getString("Guan")%>'/>
                            </div>
                            <div v-else-if="scope.row.status == '1'">
                                <img src ='${ctx}/css/images/main/monitor-ico/monitor-kpi-normalomc.png' title='<%=rb.getString("ZhengChang")%>'/>
                            </div>
                            <div v-else-if="scope.row.status == '2'">
                                <img src ='${ctx}/css/images/main/monitor-ico/monitor-kpi-brolenomc.png' title='<%=rb.getString("ChuXianYiChang")%>'/>
                            </div>
                            <div v-else >
                                <img src ='${ctx}/css/images/main/monitor-ico/monitor-kpi-brolenomc.png' title='<%=rb.getString("ChuXianYiChang")%>'/>
                            </div>
                        </template>
                    </el-table-column>
                    <el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="100"  prop="serialNumber" sortable></el-table-column>
                    <el-table-column label='<%=rb.getString("HostName")%>' min-width="100" prop="hostName" sortable></el-table-column>
                    <el-table-column label='ECI' min-width="100" prop="cellId" sortable></el-table-column>
                    <el-table-column label='<%=rb.getString("ShangBaoZhouQi")%>(<%=rb.getString("FenZhongDaXie")%>)' min-width="120" prop="reportPeriod" ></el-table-column>
                    <el-table-column label='<%=rb.getString("KaiShiShiJian")%>' min-width="100" prop="startTime" sortable></el-table-column>
                    <el-table-column label='<%=rb.getString("JieShuShiJian")%>' min-width="100" prop="endTime" sortable></el-table-column>
                </el-ctable>
            </el-tab-pane>
            
        </el-tabs>
        
        <!-- 二级页面 -- 新建 MR 任务 -->
        <el-slide ref="addMrSlide" :url="slideUrl" :title="slideTitle" :footer="slideFooter" :header='slideHeader' :position="slidePosition"
            :height="slideHeight" :modal='modal'  :width="slideWidth" :subloading="slideSubmitLoading" @ok='saveAddMrTask' @cancel='cancelAddMrSlide' 
            :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
        </el-slide>
        <!-- 二级页面 -- 下载MR文件 -->
        <el-slide  ref="sharingSlide" :url='sharingSlideUrl' :title="sharingSlideTitle" :footer="sharingSlideFooter" :header="sharingSlideHeader" :position="sharingSlidePosition"
            :height="sharingSlideHeight" :modal='modal' :width='sharingSlideWidth'  @cancel="sharingSlideCancel" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
        </el-slide>
        
        <!-- 二级页面 -- 查看MR任务结果  -->
        <el-slide ref="mrTaskResultSlide" class="mrTaskResultSlideCls" title='<%=rb.getString("JieGuo")%>' :footer="false" :header='true' position="bottom" 
            height="300px" :modal='modal'  :width="'100%'" @cancel='hideResultSlide'>
                <el-ctable id="mrTaskResultTable" ref="mrTaskResultTable" :time="6" :url="mrTaskResultTableUrl" :height="height"
                        :query-params="queryTaskResultParams"  pagination="true" :rownumber='true'>
                    
                    <!-- 主列表 -->
                    <el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="100"  prop="serial_number" ></el-table-column>
                    <el-table-column label='<%=rb.getString("HostName")%>' min-width="150" prop="host_name" ></el-table-column>
                    <el-table-column label='<%=rb.getString("ZhuangTai")%>' min-width="100" prop="progress_status" >
                        <template slot-scope="scope">
                            <div v-if="scope.row.progress_status == 'waitting'">
                                <span><%=rb.getString("DengDaiZhiXing")%></span>
                            </div>
                            <div v-if="scope.row.progress_status == 'on'">
                                <span><%=rb.getString("Kai")%></span>
                            </div>
                            <div v-if="scope.row.progress_status == 'off'">
                                <span><%=rb.getString("Guan")%></span>
                            </div>
                            <div v-if="scope.row.progress_status == 'suspend'">
                                <span><%=rb.getString("ZanTingStatus")%></span>
                            </div>
                            <div v-if="scope.row.progress_status == 'termination'">
                                <span><%=rb.getString("YiZhongZhi")%></span>
                            </div>
                        </template>
                    </el-table-column>
                    <el-table-column label='<%=rb.getString("JieGuo")%>' min-width="100" prop="progress_result" >
                        <template slot-scope="scope">
                            <div v-if="scope.row.progress_result == 'unExecuted'">
                                <span><%=rb.getString("WeiZhiXing")%></span>
                            </div>
                            <div v-if="scope.row.progress_result == 'openSuccess'">
                                <span><%=rb.getString("KaiQi")%><%=rb.getString("ChengGong")%></span>
                            </div>
                            <div v-if="scope.row.progress_result == 'openFailure'">
                                <span><%=rb.getString("KaiQi")%><%=rb.getString("ShiBai")%></span>
                            </div>
                            <div v-if="scope.row.progress_result == 'closeSuccess'">
                                <span><%=rb.getString("GuanBi")%><%=rb.getString("ChengGong")%></span>
                            </div>
                            <div v-if="scope.row.progress_result == 'closeFailure'">
                                <span><%=rb.getString("GuanBi")%><%=rb.getString("ShiBai")%></span>
                            </div>
                            <div v-if="scope.row.progress_result == 'termination'">
                                <span><%=rb.getString("YiZhongZhi")%></span>
                            </div>
                            <div v-if="scope.row.progress_result == 'reboot'">
                                <span><%=rb.getString("ChongQi")%></span>
                            </div>
                            <div v-if="scope.row.progress_result == 'rebootSuccess'">
                                <span><%=rb.getString("ChongQi")%><%=rb.getString("ChengGong")%></span>
                            </div>
                            <div v-if="scope.row.progress_result == 'rebootFailure'">
                                <span><%=rb.getString("ChongQi")%><%=rb.getString("ShiBai")%></span>
                            </div>
                        </template>
                    </el-table-column>
                    <el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' min-width="100" prop="failure_reason">
                        <template slot-scope="scope">
                            <div v-if="scope.row.failure_reason == 'rebootTimeOut'">
                                <span><%=rb.getString("ChongQiChaoShi")%></span>
                            </div>
                            <div v-else-if="scope.row.failure_reason == 'timeOut'">
                                <span><%=rb.getString("MingLingXiaFaChaoShi")%></span>
                            </div>
                            <div v-else="scope.row.failure_reason != 'rebootTimeOut' && scope.row.failure_reason != 'timeOut'">
                                <span>{{scope.row.failure_reason}}</span>
                            </div>
                        </template>
                    </el-table-column>
                    <el-table-column label='<%=rb.getString("ShiJian")%>' min-width="100" prop="run_time"></el-table-column>
                </el-ctable>
        </el-slide>
    </div>
</div>
<script type="text/javascript">
var mrPageVue = new Vue({
	el:'#mrPage',
	data:{
		activeName:'first',
        mrTaskTableUrl:'${ctx}/cell/perfmgmt/mrcustomize/getCustomizeMRTask.action',
        queryTaskParams:{
            searchText: '',
            taskStatus: '',
            taskResult: '',
            timeZone: timeZone,
            queryStartTime: '',
            queryEndTime: ''
        },
        queryTaskTime:[],
        mrTaskStatusList:[
            {value:'',text: '<%=rb.getString("QuanXuan")%>'},
            {value:'waitting',text: '<%=rb.getString("DengDaiZhiXing")%>'},
            {value:'on',text: '<%=rb.getString("JinXingZhong")%>'},
            {value:'off',text: '<%=rb.getString("YiJieShu")%>'},
            {value:'suspend',text: '<%=rb.getString("ZanTingStatus")%>'},
            {value:'termination',text: '<%=rb.getString("YiZhongZhi")%>'}
        ],
        mrTaskResultList:[
            {value:'',text: '<%=rb.getString("QuanXuan")%>'},
            {value:'success',text: '<%=rb.getString("ChengGong")%>'},
            {value:'partialSuccess',text: '<%=rb.getString("BuFenChengGong")%>'},
            {value:'failure',text: '<%=rb.getString("ShiBai")%>'},
            {value:'unExecuted',text: '<%=rb.getString("WeiZhiXing")%>'},
            {value:'inExecuted',text: '<%=rb.getString("ZhengZaiZhiXing")%>'}
        ],
        mrTaskMenuList:[],
        mrTaskResultTableUrl: '${ctx}/cell/perfmgmt/mrcustomize/getMRCellPage.action',
        queryTaskResultParams:{
            timeZone: timeZone, 
            task_id: ''
        },
        mrFileTableUrl:'${ctx}/cell/perfmgmt/mrreport/getCustomizeListPageData.action',
        queryFileParams:{
            searchText: '',
            timeZone:timeZone
        },
	    rowData:[],
	    slideUrl:'',
	    slideTitle:'',
	    slideHeader:'',
	    slideFooter:'',
	    slidePosition:'',
	    slideHeight:'',
	    slideWidth:'',
        slideSubmitLoading:false,

        sharingSlideUrl:'',
        sharingSlideTitle:'',
        sharingSlideFooter:'',
        sharingSlideHeader:'',
        sharingSlidePosition:'',
        sharingSlideHeight:'',
        sharingSlideWidth:'',
	    height:'100%',
	    width:'100%',
	    modal:false,
	},
	computed: {},
	mounted(){
        var vm = this;
		vm.init();
		eventBus.$on('hide-addSlide',vm.hideAddMrSlide);
	},
	watch:{},
	methods:{
		init(){    //根据权限判断页面默 认显示的tab内容  
			var vm = this;
		
		},
        // MR 任务列表查询
        queryMrTask(val){
            var vm = this;
            vm.queryTaskParams.searchText = val;
        },
        // MR 任务列表时间控件查询
		queryTaskTimeChange(dateTime) {
			var vm = this;
			vm.queryTaskTime = dateTime;
			if(dateTime != null){
				vm.queryTaskParams.queryStartTime = dateTime[0];
				vm.queryTaskParams.queryEndTime = dateTime[1];
			}else{
				vm.queryTaskParams.queryStartTime = '';
				vm.queryTaskParams.queryEndTime = '';
			} 
		},
        // MR 任务列表查询重置
        resetQuery(){
            var vm = this,
                params = {
                    taskStatus: '',
                    taskResult: '',
                    queryStartTime: '',
                    queryEndTime: ''
                };
            vm.queryTaskTime = [];
            Object.assign(vm.queryTaskParams,params);
            
        },
		handerClose(){ //点击页面其他地方菜单收起
	        this.$refs.mrTaskMenu.hide();
	    },
        addMrTask(){ // 新建任务
            var vm = this,
                activeName = vm.activeName;
			vm.hideResultSlide();
			vm.slideTitle = '<%=rb.getString("XinJianRenWu")%>';
	    	vm.slideHeader = true;
	    	vm.slideFooter = true;
	    	vm.slideUrl = '${ctx}/cell/perfmgmt/mrcustomize/toAddMRTaskPage.action';
	    	vm.slidePosition = 'top';
	    	vm.slideHeight = '100%';
	    	vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
			vm.$refs.addMrSlide.showSlide(function(){
                eventBus.$emit('init-addMr','','add');
				vm.modal = false;
	    	});
        },
		/**
		 * 更多操作
		 * @parame row:1.查看  2.终止  3.下载  4.删除 
		*/
		mrTaskOptClick(row,ev){ 
            var vm = this,
	    	    taskState = row.task_status;  
	    	vm.rowData = row;
            /* 菜单状态 */
            var startShow = false,
                startFlag = false,
                waitFlag = false,
                endFlag = false,
                delFlag = false,
                editFlag = false,
                startShow = false;
            
            if(taskState == 'waitting'){
                waitFlag = true;
                endFlag = true;
            }else if(taskState == 'suspend'){
                startFlag = true;
                endFlag = true;
            }else if(taskState == 'on'){
                endFlag = true;
            }else if(taskState == 'off' || taskState == 'termination'){
                delFlag = true;
            }
            if(startFlag){
                startShow = true;
                editFlag = true;
            }
			
	    	vm.mrTaskMenuList= [
		        {label:'<%=rb.getString("JieGuo")%>',cls:"el-icon el-icon-operation-result",code:'result',show:true},
                {label:'<%=rb.getString("KaiShi")%>',cls:"el-icon el-icon-operation-start CODE_PERFORMANCE_MR hidden",code:'start',show:startShow ,disable: !startFlag},
                {label:'<%=rb.getString("ZanTing")%>',cls:"el-icon el-icon-operation-awaiting CODE_PERFORMANCE_MR hidden",code:'awaiting',show: !startShow, disable: !waitFlag},
                {label:'<%=rb.getString("ZhongZhi")%>',cls:"el-icon el-icon-operation-terminate CODE_PERFORMANCE_MR hidden",code:'terminate',show: true, disable: !endFlag},
                {label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info',show: !editFlag},
                {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_PERFORMANCE_MR hidden",code:'delete',show: true, disable: !delFlag}
		    ]
	    	vm.$nextTick(function(){
	    		document.body.click();
				vm.$refs.mrTaskMenu.show(ev);
	    	});
	    },
		/**
		 *  获取点击项数据
		 * @parame ev:点击属性数据
		*/
	    mrTaskMenuClick(ev){ //单点击方法 -- 设备上报日志、告警日志 
            var vm = this,
	    	    codes = {
                    'result': vm.mrTaskViewResult,
                    'start': vm.mrTaskStart,
                    'awaiting': vm.mrTaskAwaiting,
                    'terminate': vm.mrTaskTerminate,
                    'info': vm.mrTaskInfo,
                    'delete': vm.mrTaskDelete
	    	    }
	    	if(codes[ev.code]){
	    		codes[ev.code](vm.rowData)
	    	}
	    },
        mrTaskViewResult(row){ 
            var vm = this;
            vm.queryTaskResultParams.task_id = row.task_id;
            vm.$refs.mrTaskResultSlide.showSlide(function(){
	    		vm.modal = false;
	    	});
        },
        mrTaskStart(row){ 
            var vm = this;
            vm.mrTaskProcess('start',row);
        },
        mrTaskAwaiting(row){ 
            var vm = this;
            vm.mrTaskProcess('stop',row);
        },
        mrTaskTerminate(row){ 
            var vm = this;
            vm.mrTaskProcess('terminate',row);
        },
        mrTaskInfo(row){ 
            var vm = this,
                activeName = vm.activeName;
			vm.hideResultSlide();
			vm.slideTitle = '<%=rb.getString("XinXi")%>';
	    	vm.slideHeader = true;
	    	vm.slideFooter = false;
	    	vm.slideUrl = '${ctx}/cell/perfmgmt/mrcustomize/toAddMRTaskPage.action';
	    	vm.slidePosition = 'top';
	    	vm.slideHeight = '100%';
	    	vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
			vm.$refs.addMrSlide.showSlide(function(){
                eventBus.$emit('init-addMr',row,'view');
				vm.modal = false;
	    	});
        },
        mrTaskDelete(row){ 
            var vm = this;
            vm.mrTaskProcess('del',row);
        },
        mrTaskProcess(type,row){
            var vm = this,
                codes = {
                    start: '${ctx}/cell/perfmgmt/mrcustomize/activeMRTask.action',
                    stop: '${ctx}/cell/perfmgmt/mrcustomize/suspendMRTask.action',
                    terminate: '${ctx}/cell/perfmgmt/mrcustomize/terminateMRTask.action',
                    del: '${ctx}/cell/perfmgmt/mrcustomize/clearMRTask.action'
                },
                params = {
                    timeZone: timeZone,
                    taskId: row.task_id
                };
            if(codes[type]) {
                if(type == 'del'){
                    vm.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>','<%=rb.getString("QueRen")%>',{
                        customClass:'warningConfirm',
                        confirmButtonText:'<%=rb.getString("QueDing")%>',
                        cancelButtonText:'<%=rb.getString("QuXiao")%>',
                        dangerouslyUseHTMLString:true
                    }).then(function(){
                        axios.post(codes[type],stringify(params)).then(function(response){
                            var data = response.data;
                            if(data["success"]){
                                vm.$message.success('<%=rb.getString("ChengGong")%>')
                                vm.$refs.mrTaskTableList.refresh();//表格刷新
                                vm.$refs.mrTaskResultSlide.hide(); //结果模块隐藏
                            }else{
                                vm.$message.error(data.message) //错误提示信息 
                            }
                        }) 
                    })
                }else{
                    axios.post(codes[type],stringify(params)).then(function(response){
                        var data = response.data;
                        if(data["success"]){
                            vm.$message.success('<%=rb.getString("ChengGong")%>')
                            vm.$refs.mrTaskTableList.refresh();//表格刷新
                        }else{
                            vm.$message.error(data.message) //错误提示信息 
                        }
                    }) 
                }
            }
        },
        // MR 文件列表查询
        queryMrFile(val){
            var vm = this;
            vm.queryFileParams.searchText = val;
        },
        // MR 文件列表下载
        downloadMrFile(row){
            var vm =this;
            vm.sharingSlideUrl = "${ctx}/cell/perfmgmt/mrcustomize/toDownloadMRFilePage.action";
			vm.sharingSlideHeight = '100%';
			vm.sharingSlideWidth = '100%';
			vm.sharingSlidePosition = 'top';
			vm.sharingSlideFooter = false;
			vm.sharingSlideHeader = false;
			vm.sharingSlideTitle = '';
            vm.slideSubmitLoading = false
			vm.$refs.sharingSlide.showSlide(function(){
				eventBus.$emit('mrFileDownload-init',row);
			});
        },
        // 关闭文件下载页面
		sharingSlideCancel(){
			var vm = this;
			vm.$refs.sharingSlide.hide();
			vm.$refs.mrTaskTableList.refresh();//刷新列表
		},
	    hideResultSlide(){ // 二级页面关闭
	    	var vm = this;
	    	vm.$refs.mrTaskResultSlide.hide();
	    },
		saveAddMrTask(){ //  保存新建任务
			var vm = this;
            eventBus.$emit('addMrTask-submit');
	    },
	    cancelAddMrSlide(){// 退出新建任务页面
	    	var vm = this;
	    	vm.$refs.addMrSlide.hide();
	    	vm.$refs.mrTaskTableList.refresh()
	    },
	    hideAddMrSlide(){// 关闭新建任务页面
	    	var vm = this;
	    	vm.$refs.addMrSlide.hide();
	    	vm.$refs.mrTaskTableList.refresh()
	    },
		tabClick(tab){ // 列表的点击
	    	var vm = this;

	    	vm.hideResultSlide();
	    },
	}
})
</script>